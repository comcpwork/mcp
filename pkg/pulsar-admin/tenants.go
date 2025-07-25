package pulsaradmin

import (
	"context"
	"fmt"
	"net/http"
)

// TenantsAPI 租户管理 API
type TenantsAPI struct {
	client *Client
}

// TenantInfo 租户信息
type TenantInfo struct {
	AdminRoles      []string `json:"adminRoles,omitempty"`
	AllowedClusters []string `json:"allowedClusters,omitempty"`
}

// List 获取所有租户列表
func (t *TenantsAPI) List(ctx context.Context) ([]string, error) {
	resp, err := t.client.doRequest(ctx, http.MethodGet, "/admin/v2/tenants", nil)
	if err != nil {
		return nil, err
	}

	var tenants []string
	if err := t.client.parseResponse(resp, &tenants); err != nil {
		return nil, err
	}

	return tenants, nil
}

// Get 获取指定租户信息
func (t *TenantsAPI) Get(ctx context.Context, tenant string) (*TenantInfo, error) {
	path := fmt.Sprintf("/admin/v2/tenants/%s", tenant)
	resp, err := t.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var tenantInfo TenantInfo
	if err := t.client.parseResponse(resp, &tenantInfo); err != nil {
		return nil, err
	}

	return &tenantInfo, nil
}

// Create 创建租户
func (t *TenantsAPI) Create(ctx context.Context, tenant string, tenantInfo *TenantInfo) error {
	path := fmt.Sprintf("/admin/v2/tenants/%s", tenant)
	resp, err := t.client.doRequest(ctx, http.MethodPut, path, tenantInfo)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Update 更新租户信息
func (t *TenantsAPI) Update(ctx context.Context, tenant string, tenantInfo *TenantInfo) error {
	path := fmt.Sprintf("/admin/v2/tenants/%s", tenant)
	resp, err := t.client.doRequest(ctx, http.MethodPost, path, tenantInfo)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Delete 删除租户
func (t *TenantsAPI) Delete(ctx context.Context, tenant string) error {
	path := fmt.Sprintf("/admin/v2/tenants/%s", tenant)
	resp, err := t.client.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}