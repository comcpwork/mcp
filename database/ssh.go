package database

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig SSH连接配置
type SSHConfig struct {
	Host       string // SSH服务器地址
	Port       int    // SSH端口，默认22
	User       string // SSH用户名
	Password   string // SSH密码
	KeyPath    string // 私钥文件路径
	Passphrase string // 私钥密码
}

// SSHTunnel SSH隧道，用于TCP端口转发
type SSHTunnel struct {
	config     *SSHConfig
	client     *ssh.Client
	listener   net.Listener
	localAddr  string
	remoteAddr string
	done       chan struct{}
	wg         sync.WaitGroup
}

// SSHClient SSH客户端，用于远程命令执行
type SSHClient struct {
	config *SSHConfig
	client *ssh.Client
}

// NewSSHTunnel 创建SSH隧道实例
func NewSSHTunnel(config *SSHConfig) *SSHTunnel {
	return &SSHTunnel{
		config: config,
		done:   make(chan struct{}),
	}
}

// Start 启动隧道，转发本地端口到远程地址
// remoteHost和remotePort是从SSH服务器角度能访问的目标地址
func (t *SSHTunnel) Start(remoteHost string, remotePort int) error {
	// 建立SSH连接
	client, err := createSSHClient(t.config)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	t.client = client

	// 在本地监听一个随机端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.client.Close()
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
func (t *SSHTunnel) LocalAddr() string {
	return t.localAddr
}

// Close 关闭隧道
func (t *SSHTunnel) Close() error {
	close(t.done)
	if t.listener != nil {
		t.listener.Close()
	}
	t.wg.Wait()
	if t.client != nil {
		t.client.Close()
	}
	return nil
}

// forward 转发连接
func (t *SSHTunnel) forward() {
	defer t.wg.Done()

	for {
		select {
		case <-t.done:
			return
		default:
		}

		// 设置accept超时，以便能够响应关闭信号
		t.listener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))
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

		// 建立到远程的连接
		remoteConn, err := t.client.Dial("tcp", t.remoteAddr)
		if err != nil {
			localConn.Close()
			continue
		}

		// 双向转发
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			defer localConn.Close()
			defer remoteConn.Close()

			done := make(chan struct{}, 2)

			go func() {
				io.Copy(remoteConn, localConn)
				done <- struct{}{}
			}()

			go func() {
				io.Copy(localConn, remoteConn)
				done <- struct{}{}
			}()

			// 等待任一方向完成
			<-done
		}()
	}
}

// NewSSHClient 创建SSH客户端，用于远程命令执行
func NewSSHClient(config *SSHConfig) (*SSHClient, error) {
	client, err := createSSHClient(config)
	if err != nil {
		return nil, err
	}
	return &SSHClient{
		config: config,
		client: client,
	}, nil
}

// Run 在远程服务器执行命令
func (c *SSHClient) Run(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		// 如果stderr有内容，返回stderr
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s", stderr.String())
		}
		return "", err
	}

	return stdout.String(), nil
}

// RunWithContext 在远程服务器执行命令（支持context取消）
func (c *SSHClient) RunWithContext(ctx context.Context, cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// 启动命令
	if err := session.Start(cmd); err != nil {
		return "", err
	}

	// 等待完成或context取消
	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return "", ctx.Err()
	case err := <-done:
		if err != nil {
			if stderr.Len() > 0 {
				return "", fmt.Errorf("%s", stderr.String())
			}
			return "", err
		}
		return stdout.String(), nil
	}
}

// Close 关闭SSH客户端
func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// createSSHClient 根据配置创建SSH客户端
func createSSHClient(config *SSHConfig) (*ssh.Client, error) {
	authMethods, err := getSSHAuthMethods(config)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: 可配置host key验证
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return client, nil
}

// getSSHAuthMethods 根据配置获取认证方法
func getSSHAuthMethods(config *SSHConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// 优先使用私钥认证
	if config.KeyPath != "" {
		signer, err := readPrivateKey(config.KeyPath, config.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	// 密码认证
	if config.Password != "" {
		methods = append(methods, ssh.Password(config.Password))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no SSH authentication method provided (need password or key)")
	}

	return methods, nil
}

// readPrivateKey 读取并解析私钥文件
func readPrivateKey(path string, passphrase string) (ssh.Signer, error) {
	// 展开~为home目录
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = homeDir + path[1:]
	}

	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %s: %w", path, err)
	}

	var signer ssh.Signer
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	return signer, nil
}
