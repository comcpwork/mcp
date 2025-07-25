package server

import (
	"context"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// MCPServer MCP服务器接口
type MCPServer interface {
	// Name 服务器名称
	Name() string

	// Version 服务器版本
	Version() string

	// Init 初始化服务器
	Init(ctx context.Context) error

	// RegisterTools 注册工具
	RegisterTools() error

	// RegisterResources 注册资源
	RegisterResources() error

	// GetServer 获取底层MCP服务器实例
	GetServer() *mcpserver.MCPServer

	// Start 启动服务器
	Start(ctx context.Context, transport string) error
}

// BaseServer 基础服务器实现
type BaseServer struct {
	name    string
	version string
	server  *mcpserver.MCPServer
}

// NewBaseServer 创建基础服务器
func NewBaseServer(name, version string) *BaseServer {
	return &BaseServer{
		name:    name,
		version: version,
	}
}

// Name 返回服务器名称
func (b *BaseServer) Name() string {
	return b.name
}

// Version 返回服务器版本
func (b *BaseServer) Version() string {
	return b.version
}

// GetServer 获取MCP服务器实例
func (b *BaseServer) GetServer() *mcpserver.MCPServer {
	return b.server
}

// InitMCPServer 初始化MCP服务器
func (b *BaseServer) InitMCPServer(opts ...mcpserver.ServerOption) {
	b.server = mcpserver.NewMCPServer(b.name, b.version, opts...)
}
