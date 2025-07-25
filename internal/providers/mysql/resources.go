package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mcp/internal/common"
	"mcp/pkg/log"

	"github.com/cockroachdb/errors"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/viper"
)

// handleTablesResource 处理表列表资源请求
func (s *MySQLServer) handleTablesResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// 获取数据库连接
	db, err := s.getConnection(ctx)
	if err != nil {
		if common.IsNoConfigError(err) {
			return nil, errors.New(common.FormatErrorMessage(err))
		}
		return nil, errors.Wrapf(err, "数据库连接失败")
	}
	ctx = log.WithFields(ctx,
		log.String("resource", "tables"),
		log.String("uri", req.Params.URI),
	)

	log.Info(ctx, "获取表列表资源",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldOperation, "list_tables"))

	// 获取当前数据库
	dbKey := fmt.Sprintf("databases.%s", s.activeDatabase)
	database := viper.GetString(dbKey + ".database")

	// 使用 INFORMATION_SCHEMA 查询详细表信息
	query := `
		SELECT 
			TABLE_NAME,
			TABLE_TYPE,
			ENGINE,
			TABLE_ROWS,
			DATA_LENGTH,
			INDEX_LENGTH,
			AUTO_INCREMENT,
			CREATE_TIME,
			UPDATE_TIME,
			TABLE_COMMENT
		FROM INFORMATION_SCHEMA.TABLES 
		WHERE TABLE_SCHEMA = ? 
		ORDER BY TABLE_NAME`

	rows, err := db.QueryContext(ctx, query, database)
	if err != nil {
		log.Error(ctx, "查询表列表失败",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "list_tables"),
			log.String(common.FieldError, err.Error()))
		return nil, err
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var (
			tableName, tableType, engine sql.NullString
			tableRows, dataLength, indexLength, autoIncrement sql.NullInt64
			createTime, updateTime sql.NullTime
			tableComment sql.NullString
		)
		
		if err := rows.Scan(
			&tableName, &tableType, &engine,
			&tableRows, &dataLength, &indexLength, &autoIncrement,
			&createTime, &updateTime, &tableComment,
		); err != nil {
			continue
		}
		
		// 构建表信息
		tableInfo := map[string]interface{}{
			"name":     tableName.String,
			"type":     tableType.String,
			"comment":  tableComment.String,
			"database": database,
		}
		
		// 添加可选字段
		if engine.Valid {
			tableInfo["engine"] = engine.String
		}
		if tableRows.Valid {
			// 注意：TABLE_ROWS 对于 InnoDB 表只是估算值
			tableInfo["rows_estimate"] = tableRows.Int64
		}
		if dataLength.Valid {
			tableInfo["data_size"] = dataLength.Int64
		}
		if indexLength.Valid {
			tableInfo["index_size"] = indexLength.Int64
		}
		if autoIncrement.Valid {
			tableInfo["auto_increment"] = autoIncrement.Int64
		}
		if createTime.Valid {
			tableInfo["created_at"] = createTime.Time.Format("2006-01-02 15:04:05")
		}
		if updateTime.Valid {
			tableInfo["updated_at"] = updateTime.Time.Format("2006-01-02 15:04:05")
		}

		tables = append(tables, tableInfo)
	}

	// 构建响应
	response := map[string]interface{}{
		"database": database,
		"tables":   tables,
		"count":    len(tables),
	}

	// 转换为JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	log.Info(ctx, "获取表列表资源成功",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldOperation, "list_tables"),
		log.String(common.FieldStatus, "success"),
		log.Int(common.FieldCount, len(tables)))

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}
