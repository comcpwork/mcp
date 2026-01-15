package database

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHPool SSH连接池，复用SSH连接
type SSHPool struct {
	mu      sync.RWMutex
	clients map[string]*pooledSSHClient // key: sshURI

	keepAliveInterval time.Duration // keepalive间隔
	idleTimeout       time.Duration // 空闲超时
	cleanupInterval   time.Duration // 清理间隔

	done chan struct{}
	wg   sync.WaitGroup
}

// pooledSSHClient 池化的SSH客户端
type pooledSSHClient struct {
	client   *ssh.Client
	config   *SSHConfig
	sshURI   string
	lastUsed time.Time
	healthy  bool
	done     chan struct{} // 用于停止keepalive
	mu       sync.Mutex
	refCount int // 引用计数
}

// 全局连接池实例
var (
	globalPool     *SSHPool
	globalPoolOnce sync.Once
)

// GetSSHPool 获取全局SSH连接池
func GetSSHPool() *SSHPool {
	globalPoolOnce.Do(func() {
		globalPool = NewSSHPool(
			30*time.Second, // keepalive间隔
			5*time.Minute,  // 空闲超时
			1*time.Minute,  // 清理间隔
		)
	})
	return globalPool
}

// NewSSHPool 创建新的SSH连接池
func NewSSHPool(keepAliveInterval, idleTimeout, cleanupInterval time.Duration) *SSHPool {
	pool := &SSHPool{
		clients:           make(map[string]*pooledSSHClient),
		keepAliveInterval: keepAliveInterval,
		idleTimeout:       idleTimeout,
		cleanupInterval:   cleanupInterval,
		done:              make(chan struct{}),
	}

	// 启动清理goroutine
	pool.wg.Add(1)
	go pool.cleanupLoop()

	return pool
}

// GetClient 从池中获取SSH客户端，如果不存在则创建
func (p *SSHPool) GetClient(sshURI string) (*ssh.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查是否已有连接
	if pc, ok := p.clients[sshURI]; ok {
		pc.mu.Lock()
		if pc.healthy && pc.client != nil {
			pc.lastUsed = time.Now()
			pc.refCount++
			pc.mu.Unlock()
			return pc.client, nil
		}
		pc.mu.Unlock()
		// 连接不健康，关闭并移除
		p.removeClientLocked(sshURI)
	}

	// 创建新连接
	sshConfig, err := ParseSSHURI(sshURI)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH URI: %w", err)
	}

	client, err := createSSHClient(sshConfig)
	if err != nil {
		return nil, err
	}

	// 创建池化客户端
	pc := &pooledSSHClient{
		client:   client,
		config:   sshConfig,
		sshURI:   sshURI,
		lastUsed: time.Now(),
		healthy:  true,
		done:     make(chan struct{}),
		refCount: 1,
	}

	// 启动keepalive
	p.wg.Add(1)
	go p.keepAlive(pc)

	p.clients[sshURI] = pc
	return client, nil
}

// ReleaseClient 释放SSH客户端引用（不关闭连接）
func (p *SSHPool) ReleaseClient(sshURI string) {
	p.mu.RLock()
	pc, ok := p.clients[sshURI]
	p.mu.RUnlock()

	if ok {
		pc.mu.Lock()
		if pc.refCount > 0 {
			pc.refCount--
		}
		pc.lastUsed = time.Now()
		pc.mu.Unlock()
	}
}

// GetTunnel 获取SSH隧道（复用池中的SSH连接）
func (p *SSHPool) GetTunnel(sshURI string, remoteHost string, remotePort int) (*PooledSSHTunnel, error) {
	client, err := p.GetClient(sshURI)
	if err != nil {
		return nil, err
	}

	tunnel := &PooledSSHTunnel{
		pool:   p,
		sshURI: sshURI,
		client: client,
		done:   make(chan struct{}),
	}

	if err := tunnel.start(remoteHost, remotePort); err != nil {
		p.ReleaseClient(sshURI)
		return nil, err
	}

	return tunnel, nil
}

// GetSSHExecClient 获取用于远程命令执行的SSH客户端
func (p *SSHPool) GetSSHExecClient(sshURI string) (*PooledSSHExecClient, error) {
	client, err := p.GetClient(sshURI)
	if err != nil {
		return nil, err
	}

	return &PooledSSHExecClient{
		pool:   p,
		sshURI: sshURI,
		client: client,
	}, nil
}

// keepAlive 保持连接活跃
func (p *SSHPool) keepAlive(pc *pooledSSHClient) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.keepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pc.mu.Lock()
			if !pc.healthy || pc.client == nil {
				pc.mu.Unlock()
				return
			}
			client := pc.client
			pc.mu.Unlock()

			// 发送keepalive请求
			// 使用openssh的请求名，大多数服务器都支持
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				pc.mu.Lock()
				pc.healthy = false
				pc.mu.Unlock()
				return
			}

		case <-pc.done:
			return

		case <-p.done:
			return
		}
	}
}

// cleanupLoop 定期清理空闲连接
func (p *SSHPool) cleanupLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanup()
		case <-p.done:
			return
		}
	}
}

// cleanup 清理空闲和不健康的连接
func (p *SSHPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for uri, pc := range p.clients {
		pc.mu.Lock()
		// 清理不健康或空闲超时且无引用的连接
		shouldRemove := !pc.healthy ||
			(pc.refCount == 0 && now.Sub(pc.lastUsed) > p.idleTimeout)
		pc.mu.Unlock()

		if shouldRemove {
			p.removeClientLocked(uri)
		}
	}
}

// removeClientLocked 移除客户端（调用者需持有锁）
func (p *SSHPool) removeClientLocked(sshURI string) {
	if pc, ok := p.clients[sshURI]; ok {
		close(pc.done)
		if pc.client != nil {
			pc.client.Close()
		}
		delete(p.clients, sshURI)
	}
}

// Close 关闭连接池
func (p *SSHPool) Close() {
	close(p.done)

	p.mu.Lock()
	for uri := range p.clients {
		p.removeClientLocked(uri)
	}
	p.mu.Unlock()

	p.wg.Wait()
}

// PooledSSHTunnel 池化的SSH隧道
type PooledSSHTunnel struct {
	pool       *SSHPool
	sshURI     string
	client     *ssh.Client
	listener   net.Listener
	localAddr  string
	remoteAddr string
	done       chan struct{}
	wg         sync.WaitGroup
}

// start 启动隧道
func (t *PooledSSHTunnel) start(remoteHost string, remotePort int) error {
	// 在本地监听一个随机端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to create local listener: %w", err)
	}
	t.listener = listener
	t.localAddr = listener.Addr().String()
	t.remoteAddr = fmt.Sprintf("%s:%d", remoteHost, remotePort)

	// 启动转发goroutine
	t.wg.Add(1)
	go t.forward()

	return nil
}

// LocalAddr 获取本地隧道地址
func (t *PooledSSHTunnel) LocalAddr() string {
	return t.localAddr
}

// Close 关闭隧道（但不关闭SSH连接）
func (t *PooledSSHTunnel) Close() error {
	close(t.done)
	if t.listener != nil {
		t.listener.Close()
	}
	t.wg.Wait()
	// 释放SSH连接引用
	t.pool.ReleaseClient(t.sshURI)
	return nil
}

// dialWithTimeout 带超时的 SSH Dial
func (t *PooledSSHTunnel) dialWithTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	type dialResult struct {
		conn net.Conn
		err  error
	}

	result := make(chan dialResult, 1)
	go func() {
		conn, err := t.client.Dial(network, addr)
		result <- dialResult{conn, err}
	}()

	select {
	case r := <-result:
		return r.conn, r.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("dial timeout after %v", timeout)
	case <-t.done:
		return nil, fmt.Errorf("tunnel closed")
	}
}

// forward 转发连接
func (t *PooledSSHTunnel) forward() {
	defer t.wg.Done()

	for {
		select {
		case <-t.done:
			return
		default:
		}

		// 设置accept超时，以便能够响应关闭信号
		// 使用较短的超时以减少延迟
		if tcpListener, ok := t.listener.(*net.TCPListener); ok {
			tcpListener.SetDeadline(time.Now().Add(100 * time.Millisecond))
		}

		localConn, err := t.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-t.done:
				return
			default:
				continue
			}
		}

		// 建立到远程的连接（带超时）
		remoteConn, err := t.dialWithTimeout("tcp", t.remoteAddr, 30*time.Second)
		if err != nil {
			localConn.Close()
			continue
		}

		// 双向转发
		t.wg.Add(1)
		go func(local net.Conn, remote net.Conn) {
			defer t.wg.Done()
			defer local.Close()
			defer remote.Close()

			done := make(chan struct{}, 2)

			go func() {
				io.Copy(remote, local)
				done <- struct{}{}
			}()

			go func() {
				io.Copy(local, remote)
				done <- struct{}{}
			}()

			// 等待任一方向完成
			<-done
		}(localConn, remoteConn)
	}
}

// PooledSSHExecClient 池化的SSH执行客户端
type PooledSSHExecClient struct {
	pool   *SSHPool
	sshURI string
	client *ssh.Client
}

// Run 执行远程命令
func (c *PooledSSHExecClient) Run(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		if len(output) > 0 {
			return "", fmt.Errorf("%s", string(output))
		}
		return "", err
	}

	return string(output), nil
}

// RunWithContext 执行远程命令（支持context取消）
func (c *PooledSSHExecClient) RunWithContext(ctx context.Context, cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// 启动命令
	output := make(chan struct {
		result string
		err    error
	}, 1)

	go func() {
		out, err := session.CombinedOutput(cmd)
		if err != nil {
			if len(out) > 0 {
				output <- struct {
					result string
					err    error
				}{"", fmt.Errorf("%s", string(out))}
				return
			}
			output <- struct {
				result string
				err    error
			}{"", err}
			return
		}
		output <- struct {
			result string
			err    error
		}{string(out), nil}
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return "", ctx.Err()
	case res := <-output:
		return res.result, res.err
	}
}

// Close 释放引用（不关闭SSH连接）
func (c *PooledSSHExecClient) Close() error {
	c.pool.ReleaseClient(c.sshURI)
	return nil
}
