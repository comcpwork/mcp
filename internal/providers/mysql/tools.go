package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"mcp/internal/common"
	"mcp/pkg/log"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/xwb1989/sqlparser"
	"github.com/spf13/viper"
)

// ensureConnection 获取数据库连接，如果失败返回错误结果
func (s *MySQLServer) ensureConnection(ctx context.Context) (*sql.DB, *mcp.CallToolResult) {
	log.Info(ctx, "尝试获取数据库连接",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldInstance, s.activeDatabase))
	
	db, err := s.getConnection(ctx)
	if err != nil {
		log.Error(ctx, "数据库连接失败",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldInstance, s.activeDatabase),
			log.String(common.FieldError, err.Error()))
		
		if common.IsNoConfigError(err) {
			return nil, mcp.NewToolResultError(common.FormatErrorMessage(err))
		}
		return nil, mcp.NewToolResultError(fmt.Sprintf("数据库连接失败: %v", err))
	}
	
	// 连接成功，记录连接信息
	dbKey := fmt.Sprintf("databases.%s", s.activeDatabase)
	host := viper.GetString(dbKey + ".host")
	port := viper.GetInt(dbKey + ".port")
	
	log.Info(ctx, "数据库连接成功",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldInstance, s.activeDatabase),
		log.String(common.FieldHost, host),
		log.Int(common.FieldPort, port),
		log.String(common.FieldStatus, "connected"))
	
	return db, nil
}

// handleExec 处理SQL执行请求
func (s *MySQLServer) handleExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取参数
	sqlQuery, err := req.RequireString("sql")
	if err != nil {
		return nil, errors.Wrap(err, "缺少sql参数")
	}

	// 获取limit参数，优先使用传入的limit，然后是配置文件中的max_rows，最后使用默认值
	configMaxRows := viper.GetInt("tools.query.max_rows")
	if configMaxRows == 0 {
		configMaxRows = common.DefaultMaxRows // 使用常量替代魔法数字
	}
	limit := req.GetInt("limit", configMaxRows)

	// 记录查询开始
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "exec"),
		log.String(common.FieldDatabase, s.activeDatabase))
	
	log.Info(ctx, "处理查询请求",
		log.String(common.FieldSQL, sqlQuery),
		log.Int("limit", limit))

	// 获取数据库连接
	db, errResult := s.ensureConnection(ctx)
	if errResult != nil {
		return errResult, nil
	}

	// 使用SQL解析器进行精确的安全检查
	if err := s.validateSQLSecurity(ctx, sqlQuery); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 使用SQL解析器判断是查询还是执行操作
	isQuery := s.isQueryStatement(ctx, sqlQuery)

	if isQuery {
		// 执行查询操作
		return s.executeQuery(ctx, db, sqlQuery, limit)
	} else {
		// 执行修改操作（INSERT/UPDATE/DELETE等）
		return s.executeModification(ctx, db, sqlQuery)
	}
}

// handleShowTables 显示所有表
func (s *MySQLServer) handleShowTables(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	database := req.GetString("database", viper.GetString("database.database"))
	
	log.Info(ctx, "处理 show_tables 请求",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "show_tables"),
		log.String(common.FieldOperation, "list_tables"),
		log.String(common.FieldDatabase, database),
		log.String("active_database", s.activeDatabase),
	)
	// 获取数据库连接
	db, errResult := s.ensureConnection(ctx)
	if errResult != nil {
		return errResult, nil
	}

	// 获取当前数据库名（如果没有指定database参数）
	if database == "" {
		// 从活跃数据库配置中获取数据库名
		dbKey := fmt.Sprintf("databases.%s.database", s.activeDatabase)
		database = viper.GetString(dbKey)
	}
	
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
	
	log.Info(ctx, "开始查询表详细信息", 
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldOperation, "query_schema"),
		log.String("query", "INFORMATION_SCHEMA.TABLES"),
		log.String("schema", database),
	)

	rows, err := db.QueryContext(ctx, query, database)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询表列表失败: %v", err)), nil
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
			"name":    tableName.String,
			"type":    tableType.String,
			"comment": tableComment.String,
		}
		
		// 添加可选字段（如果有值的话）
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

	// 使用紧凑输出格式
	result := s.formatShowTablesCompact(tables, database)

	log.Info(ctx, "获取表列表成功",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "show_tables"),
		log.String(common.FieldOperation, "list_tables"),
		log.String(common.FieldStatus, "success"),
		log.String(common.FieldDatabase, database),
		log.Int(common.FieldCount, len(tables)),
	)
	return mcp.NewToolResultText(result), nil
}

// handleDescribeTable 描述表结构
func (s *MySQLServer) handleDescribeTable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tableName, err := req.RequireString("table")
	if err != nil {
		return nil, errors.Wrap(err, "缺少table参数")
	}

	log.Info(ctx, "处理 describe_table 请求",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "describe_table"),
		log.String(common.FieldOperation, "describe"),
		log.String(common.FieldTable, tableName),
		log.String("active_database", s.activeDatabase),
	)

	// 获取数据库连接
	db, errResult := s.ensureConnection(ctx)
	if errResult != nil {
		return errResult, nil
	}

	log.Info(ctx, "获取表结构",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "describe_table"),
		log.String(common.FieldOperation, "describe"),
		log.String(common.FieldTable, tableName))

	// 查询表结构（包含注释信息）
	query := fmt.Sprintf(`
		SELECT 
			COLUMN_NAME as field,
			COLUMN_TYPE as type,
			IS_NULLABLE as nullable,
			COLUMN_KEY as key_type,
			COLUMN_DEFAULT as default_value,
			EXTRA as extra,
			COLUMN_COMMENT as comment
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '%s'
		ORDER BY ORDINAL_POSITION
	`, tableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询表结构失败: %v", err)), nil
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var field, typ, nullable, keyType, defaultVal, extra, comment sql.NullString
		if err := rows.Scan(&field, &typ, &nullable, &keyType, &defaultVal, &extra, &comment); err != nil {
			continue
		}

		column := map[string]interface{}{
			"field":   field.String,
			"type":    typ.String,
			"null":    nullable.String == "YES",
			"key":     keyType.String,
			"default": defaultVal.String,
			"extra":   extra.String,
			"comment": comment.String,
		}
		columns = append(columns, column)
	}

	// 根据输出格式生成响应
	// 使用紧凑输出格式
	result := s.formatDescribeTableCompact(tableName, columns)

	log.Info(ctx, "获取表结构成功",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "describe_table"),
		log.String(common.FieldOperation, "describe"),
		log.String(common.FieldStatus, "success"),
		log.String(common.FieldTable, tableName),
		log.Int("columns", len(columns)))
	return mcp.NewToolResultText(result), nil
}

// handleDescribeTables 描述多个表的结构
func (s *MySQLServer) handleDescribeTables(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tablesParam, err := req.RequireString("tables")
	if err != nil {
		return nil, errors.Wrap(err, "缺少tables参数")
	}

	includeIndexes := req.GetBool("include_indexes", false)
	includeForeignKeys := req.GetBool("include_foreign_keys", false)

	log.Info(ctx, "处理 describe_tables 请求",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "describe_tables"),
		log.String(common.FieldOperation, "describe_multiple"),
		log.String("tables", tablesParam),
		log.Bool("include_indexes", includeIndexes),
		log.Bool("include_foreign_keys", includeForeignKeys),
		log.String("active_database", s.activeDatabase),
	)

	// 获取数据库连接
	db, errResult := s.ensureConnection(ctx)
	if errResult != nil {
		return errResult, nil
	}

	// 解析表名列表
	tableNames := strings.Split(strings.TrimSpace(tablesParam), ",")
	var cleanTableNames []string
	for _, name := range tableNames {
		cleanName := strings.TrimSpace(name)
		if cleanName != "" {
			cleanTableNames = append(cleanTableNames, cleanName)
		}
	}

	if len(cleanTableNames) == 0 {
		return mcp.NewToolResultError("没有提供有效的表名"), nil
	}

	log.Info(ctx, "开始查询多个表结构", 
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldOperation, "batch_describe"),
		log.Int("table_count", len(cleanTableNames)),
		log.Any("table_names", cleanTableNames))

	// 获取当前数据库名
	currentDB := viper.GetString(fmt.Sprintf("databases.%s.database", s.activeDatabase))

	var tablesInfo []map[string]interface{}

	for _, tableName := range cleanTableNames {
		log.Info(ctx, "查询表结构",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "describe_single"),
			log.String(common.FieldTable, tableName))
		
		tableInfo := map[string]interface{}{
			"table_name": tableName,
		}

		// 1. 获取基本表信息
		basicInfo, err := s.getTableBasicInfo(ctx, db, tableName, currentDB)
		if err != nil {
			log.Error(ctx, "获取表基本信息失败",
				log.String(common.FieldProvider, "mysql"),
				log.String(common.FieldOperation, "get_table_info"),
				log.String(common.FieldTable, tableName),
				log.String(common.FieldError, err.Error()))
			tableInfo["error"] = fmt.Sprintf("获取表 %s 基本信息失败: %v", tableName, err)
			tablesInfo = append(tablesInfo, tableInfo)
			continue
		}
		
		// 合并基本信息
		for k, v := range basicInfo {
			tableInfo[k] = v
		}

		// 2. 获取列信息
		columns, err := s.getTableColumns(ctx, db, tableName)
		if err != nil {
			log.Error(ctx, "获取表列信息失败",
				log.String(common.FieldProvider, "mysql"),
				log.String(common.FieldOperation, "get_columns"),
				log.String(common.FieldTable, tableName),
				log.String(common.FieldError, err.Error()))
			tableInfo["error"] = fmt.Sprintf("获取表 %s 列信息失败: %v", tableName, err)
			tablesInfo = append(tablesInfo, tableInfo)
			continue
		}
		tableInfo["columns"] = columns
		tableInfo["column_count"] = len(columns)

		// 3. 获取索引信息（可选）
		if includeIndexes {
			indexes, err := s.getTableIndexes(ctx, db, tableName, currentDB)
			if err != nil {
				log.Error(ctx, "获取表索引信息失败",
					log.String(common.FieldProvider, "mysql"),
					log.String(common.FieldOperation, "get_indexes"),
					log.String(common.FieldTable, tableName),
					log.String(common.FieldError, err.Error()))
				tableInfo["index_error"] = fmt.Sprintf("获取索引信息失败: %v", err)
			} else {
				tableInfo["indexes"] = indexes
				tableInfo["index_count"] = len(indexes)
			}
		}

		// 4. 获取外键信息（可选）
		if includeForeignKeys {
			foreignKeys, err := s.getTableForeignKeys(ctx, db, tableName, currentDB)
			if err != nil {
				log.Error(ctx, "获取表外键信息失败",
					log.String(common.FieldProvider, "mysql"),
					log.String(common.FieldOperation, "get_foreign_keys"),
					log.String(common.FieldTable, tableName),
					log.String(common.FieldError, err.Error()))
				tableInfo["foreign_key_error"] = fmt.Sprintf("获取外键信息失败: %v", err)
			} else {
				tableInfo["foreign_keys"] = foreignKeys
				tableInfo["foreign_key_count"] = len(foreignKeys)
			}
		}

		tablesInfo = append(tablesInfo, tableInfo)
		log.Info(ctx, "表结构查询完成",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "describe_single"),
			log.String(common.FieldStatus, "success"),
			log.String(common.FieldTable, tableName))
	}

	// 根据输出格式生成响应
	// 使用紧凑输出格式
	result := s.formatTablesCompact(tablesInfo, currentDB, includeIndexes, includeForeignKeys)

	log.Info(ctx, "获取多个表结构成功", 
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "describe_tables"),
		log.String(common.FieldOperation, "describe_multiple"),
		log.String(common.FieldStatus, "success"),
		log.Int("total_tables", len(tablesInfo)),
		log.Int("requested_tables", len(cleanTableNames)))
	return mcp.NewToolResultText(result), nil
}

// handleListDatabases 列出所有数据库配置
// getTableBasicInfo 获取表的基本信息
func (s *MySQLServer) getTableBasicInfo(ctx context.Context, db *sql.DB, tableName, database string) (map[string]interface{}, error) {
	query := `
		SELECT 
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
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`

	var (
		tableType, engine sql.NullString
		tableRows, dataLength, indexLength, autoIncrement sql.NullInt64
		createTime, updateTime sql.NullTime
		tableComment sql.NullString
	)

	err := db.QueryRowContext(ctx, query, database, tableName).Scan(
		&tableType, &engine, &tableRows, &dataLength, &indexLength,
		&autoIncrement, &createTime, &updateTime, &tableComment,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Newf("表 %s 不存在", tableName)
		}
		return nil, err
	}

	info := make(map[string]interface{})
	
	if tableType.Valid {
		info["table_type"] = tableType.String
	}
	if engine.Valid {
		info["engine"] = engine.String
	}
	if tableRows.Valid {
		info["rows_estimate"] = tableRows.Int64
	}
	if dataLength.Valid {
		info["data_size"] = dataLength.Int64
	}
	if indexLength.Valid {
		info["index_size"] = indexLength.Int64
	}
	if autoIncrement.Valid {
		info["auto_increment"] = autoIncrement.Int64
	}
	if createTime.Valid {
		info["created_at"] = createTime.Time.Format("2006-01-02 15:04:05")
	}
	if updateTime.Valid {
		info["updated_at"] = updateTime.Time.Format("2006-01-02 15:04:05")
	}
	if tableComment.Valid {
		info["comment"] = tableComment.String
	}

	return info, nil
}

// getTableColumns 获取表的列信息（包含注释）
func (s *MySQLServer) getTableColumns(ctx context.Context, db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT 
			COLUMN_NAME as field,
			COLUMN_TYPE as type,
			IS_NULLABLE as nullable,
			COLUMN_KEY as key_type,
			COLUMN_DEFAULT as default_value,
			EXTRA as extra,
			COLUMN_COMMENT as comment
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '%s'
		ORDER BY ORDINAL_POSITION
	`, tableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var field, typ, nullable, keyType, defaultVal, extra, comment sql.NullString
		if err := rows.Scan(&field, &typ, &nullable, &keyType, &defaultVal, &extra, &comment); err != nil {
			continue
		}

		column := map[string]interface{}{
			"field":   field.String,
			"type":    typ.String,
			"null":    nullable.String == "YES",
			"key":     keyType.String,
			"default": defaultVal.String,
			"extra":   extra.String,
			"comment": comment.String,
		}
		columns = append(columns, column)
	}

	return columns, nil
}

// getTableIndexes 获取表的索引信息
func (s *MySQLServer) getTableIndexes(ctx context.Context, db *sql.DB, tableName, database string) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			SEQ_IN_INDEX,
			COLLATION,
			CARDINALITY,
			SUB_PART,
			PACKED,
			NULLABLE,
			INDEX_TYPE,
			COMMENT
		FROM INFORMATION_SCHEMA.STATISTICS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX`

	rows, err := db.QueryContext(ctx, query, database, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]map[string]interface{})
	
	for rows.Next() {
		var (
			indexName, columnName sql.NullString
			nonUnique, seqInIndex sql.NullInt64
			collation, packed, nullable, indexType, comment sql.NullString
			cardinality, subPart sql.NullInt64
		)

		err := rows.Scan(
			&indexName, &columnName, &nonUnique, &seqInIndex,
			&collation, &cardinality, &subPart, &packed,
			&nullable, &indexType, &comment,
		)
		if err != nil {
			continue
		}

		if !indexName.Valid {
			continue
		}

		idxName := indexName.String
		if _, exists := indexMap[idxName]; !exists {
			indexMap[idxName] = map[string]interface{}{
				"index_name": idxName,
				"unique":     nonUnique.Valid && nonUnique.Int64 == 0,
				"type":       indexType.String,
				"comment":    comment.String,
				"columns":    make([]map[string]interface{}, 0),
			}
		}

		// 添加列信息
		columnInfo := map[string]interface{}{
			"column_name":  columnName.String,
			"seq_in_index": seqInIndex.Int64,
		}
		if collation.Valid {
			columnInfo["collation"] = collation.String
		}
		if cardinality.Valid {
			columnInfo["cardinality"] = cardinality.Int64
		}
		if subPart.Valid {
			columnInfo["sub_part"] = subPart.Int64
		}

		columns := indexMap[idxName]["columns"].([]map[string]interface{})
		indexMap[idxName]["columns"] = append(columns, columnInfo)
	}

	// 转换为数组
	var indexes []map[string]interface{}
	for _, index := range indexMap {
		indexes = append(indexes, index)
	}

	return indexes, nil
}

// getTableForeignKeys 获取表的外键信息
func (s *MySQLServer) getTableForeignKeys(ctx context.Context, db *sql.DB, tableName, database string) ([]map[string]interface{}, error) {
	// 先从 KEY_COLUMN_USAGE 获取基本外键信息
	query1 := `
		SELECT 
			CONSTRAINT_NAME,
			COLUMN_NAME,
			REFERENCED_TABLE_SCHEMA,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
		WHERE TABLE_SCHEMA = ? 
			AND TABLE_NAME = ? 
			AND REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY CONSTRAINT_NAME, ORDINAL_POSITION`

	rows, err := db.QueryContext(ctx, query1, database, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[string]map[string]interface{})
	
	for rows.Next() {
		var (
			constraintName, columnName sql.NullString
			referencedSchema, referencedTable, referencedColumn sql.NullString
		)

		err := rows.Scan(
			&constraintName, &columnName, &referencedSchema,
			&referencedTable, &referencedColumn,
		)
		if err != nil {
			continue
		}

		if !constraintName.Valid {
			continue
		}

		fkName := constraintName.String
		if _, exists := fkMap[fkName]; !exists {
			fkMap[fkName] = map[string]interface{}{
				"constraint_name":      fkName,
				"referenced_schema":    referencedSchema.String,
				"referenced_table":     referencedTable.String,
				"update_rule":          "UNKNOWN",
				"delete_rule":          "UNKNOWN",
				"column_mappings":      make([]map[string]interface{}, 0),
			}
		}

		// 添加列映射
		columnMapping := map[string]interface{}{
			"column_name":            columnName.String,
			"referenced_column_name": referencedColumn.String,
		}

		mappings := fkMap[fkName]["column_mappings"].([]map[string]interface{})
		fkMap[fkName]["column_mappings"] = append(mappings, columnMapping)
	}

	// 尝试获取更详细的外键信息（如果支持的话）
	query2 := `
		SELECT 
			CONSTRAINT_NAME,
			UPDATE_RULE,
			DELETE_RULE
		FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS 
		WHERE CONSTRAINT_SCHEMA = ?
			AND TABLE_NAME = ?`

	rows2, err := db.QueryContext(ctx, query2, database, tableName)
	if err == nil {
		defer rows2.Close()
		
		for rows2.Next() {
			var (
				constraintName sql.NullString
				updateRule, deleteRule sql.NullString
			)

			err := rows2.Scan(&constraintName, &updateRule, &deleteRule)
			if err != nil {
				continue
			}

			if constraintName.Valid && fkMap[constraintName.String] != nil {
				if updateRule.Valid {
					fkMap[constraintName.String]["update_rule"] = updateRule.String
				}
				if deleteRule.Valid {
					fkMap[constraintName.String]["delete_rule"] = deleteRule.String
				}
			}
		}
	}

	// 转换为数组
	var foreignKeys []map[string]interface{}
	for _, fk := range fkMap {
		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

// formatTablesCompact 生成紧凑格式输出，节省token
func (s *MySQLServer) formatTablesCompact(tablesInfo []map[string]interface{}, database string, includeIndexes, includeForeignKeys bool) string {
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("Database: %s (%d tables)\n\n", database, len(tablesInfo)))
	
	for i, table := range tablesInfo {
		tableName := table["table_name"].(string)
		
		// 检查是否有错误
		if errorMsg, hasError := table["error"]; hasError {
			result.WriteString(fmt.Sprintf("%d. %s - Error: %s\n\n", i+1, tableName, errorMsg))
			continue
		}
		
		// 表基本信息
		result.WriteString(fmt.Sprintf("%d. %s", i+1, tableName))
		
		if engine, ok := table["engine"]; ok {
			result.WriteString(fmt.Sprintf(" (%s)", engine))
		}
		
		if comment, ok := table["comment"]; ok && comment.(string) != "" {
			result.WriteString(fmt.Sprintf(" - %s", comment))
		}
		
		result.WriteString("\n")
		
		// 表统计信息（一行显示）
		var stats []string
		if rows, ok := table["rows_estimate"]; ok {
			stats = append(stats, fmt.Sprintf("rows≈%v", rows))
		}
		if dataSize, ok := table["data_size"]; ok {
			stats = append(stats, fmt.Sprintf("data %s", s.formatBytes(dataSize.(int64))))
		}
		if indexSize, ok := table["index_size"]; ok {
			stats = append(stats, fmt.Sprintf("index %s", s.formatBytes(indexSize.(int64))))
		}
		if len(stats) > 0 {
			result.WriteString(fmt.Sprintf("   Stats: %s\n", strings.Join(stats, ", ")))
		}
		
		// 列信息（紧凑格式）
		if columns, ok := table["columns"]; ok {
			result.WriteString("   Columns: ")
			columnList := columns.([]map[string]interface{})
			var columnStrs []string
			
			for _, col := range columnList {
				field := col["field"].(string)
				typ := col["type"].(string)
				
				// 构建列描述
				colStr := fmt.Sprintf("%s:%s", field, typ)
				
				// 添加关键信息
				var attrs []string
				if key := col["key"].(string); key != "" {
					switch key {
					case "PRI":
						attrs = append(attrs, "PK")
					case "UNI":
						attrs = append(attrs, "UNIQUE")
					case "MUL":
						attrs = append(attrs, "INDEX")
					}
				}
				
				if !col["null"].(bool) {
					attrs = append(attrs, "NOT NULL")
				}
				
				if extra := col["extra"].(string); extra != "" {
					if strings.Contains(extra, "auto_increment") {
						attrs = append(attrs, "AUTO_INC")
					}
				}
				
				if len(attrs) > 0 {
					colStr += fmt.Sprintf("[%s]", strings.Join(attrs, ","))
				}
				
				columnStrs = append(columnStrs, colStr)
			}
			result.WriteString(strings.Join(columnStrs, ", "))
			result.WriteString("\n")
		}
		
		// 索引信息（紧凑格式）
		if includeIndexes {
			if indexes, ok := table["indexes"]; ok && indexes != nil {
				indexList := indexes.([]map[string]interface{})
				if len(indexList) > 0 {
					result.WriteString("   Indexes: ")
					var indexStrs []string
					
					for _, idx := range indexList {
						indexName := idx["index_name"].(string)
						unique := idx["unique"].(bool)
						columns := idx["columns"].([]map[string]interface{})
						
						var columnNames []string
						for _, col := range columns {
							columnNames = append(columnNames, col["column_name"].(string))
						}
						
						indexStr := fmt.Sprintf("%s(%s)", indexName, strings.Join(columnNames, ","))
						if unique {
							indexStr += "[UNIQUE]"
						}
						
						indexStrs = append(indexStrs, indexStr)
					}
					result.WriteString(strings.Join(indexStrs, ", "))
					result.WriteString("\n")
				}
			}
		}
		
		// 外键信息（紧凑格式）
		if includeForeignKeys {
			if foreignKeys, ok := table["foreign_keys"]; ok && foreignKeys != nil {
				fkList := foreignKeys.([]map[string]interface{})
				if len(fkList) > 0 {
					result.WriteString("   外键: ")
					var fkStrs []string
					
					for _, fk := range fkList {
						constraintName := fk["constraint_name"].(string)
						referencedTable := fk["referenced_table"].(string)
						columnMappings := fk["column_mappings"].([]map[string]interface{})
						
						var mappingStrs []string
						for _, mapping := range columnMappings {
							mappingStrs = append(mappingStrs, fmt.Sprintf("%s→%s.%s", 
								mapping["column_name"], 
								referencedTable,
								mapping["referenced_column_name"]))
						}
						
						fkStr := fmt.Sprintf("%s(%s)", constraintName, strings.Join(mappingStrs, ","))
						fkStrs = append(fkStrs, fkStr)
					}
					result.WriteString(strings.Join(fkStrs, ", "))
					result.WriteString("\n")
				}
			}
		}
		
		result.WriteString("\n")
	}
	
	return result.String()
}

// formatBytes 格式化字节大小
func (s *MySQLServer) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// executeQuery 执行查询操作
func (s *MySQLServer) executeQuery(ctx context.Context, db *sql.DB, sqlQuery string, limit int) (*mcp.CallToolResult, error) {
	// 记录查询开始
	log.Info(ctx, "执行SQL查询",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "exec"),
		log.String(common.FieldOperation, "query"),
		log.String(common.FieldSQL, sqlQuery),
		log.Int("limit", limit))
	
	startTime := time.Now()
	
	rows, err := db.QueryContext(ctx, sqlQuery)
	if err != nil {
		log.Error(ctx, "查询执行失败",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldTool, "exec"),
			log.String(common.FieldOperation, "query"),
			log.String(common.FieldSQL, sqlQuery),
			log.String(common.FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		log.Error(ctx, "获取列信息失败",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "get_columns"),
			log.String(common.FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("获取列信息失败: %v", err)), nil
	}
	log.Info(ctx, "获取列信息成功",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldOperation, "get_columns"),
		log.Int("column_count", len(columns)),
		log.Any("columns", columns))

	// 读取结果 - 初始化为空数组而不是nil，确保JSON序列化时返回[]而不是null
	results := make([]map[string]interface{}, 0)
	count := 0
	rowNum := 0

	for rows.Next() && count < limit {
		rowNum++
		log.Info(ctx, "开始处理行",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "scan_row"),
			log.Int("row_number", rowNum))
		
		// 创建一个切片来存储列值
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		// 扫描行
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Error(ctx, "扫描行失败",
				log.String(common.FieldProvider, "mysql"),
				log.String(common.FieldOperation, "scan_row"),
				log.Int("row_number", rowNum),
				log.String(common.FieldError, err.Error()))
			continue
		}
		log.Info(ctx, "行扫描成功",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "scan_row"),
			log.Int("row_number", rowNum))

		// 构建结果map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// 处理空值
			if val == nil {
				row[col] = nil
			} else {
				// 处理字节数组（通常是字符串）
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}
		}

		log.Info(ctx, "行数据构建完成", 
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "build_row"),
			log.Int("row_number", rowNum), 
			log.String("row_data", fmt.Sprintf("%+v", row)))

		results = append(results, row)
		count++
	}
	
	// 检查是否有扫描后的错误
	if err := rows.Err(); err != nil {
		log.Error(ctx, "遍历结果时发生错误",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "iterate_rows"),
			log.String(common.FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("遍历结果失败: %v", err)), nil
	}
	
	// 记录查询结果
	duration := time.Since(startTime).Milliseconds()
	log.Info(ctx, "查询完成",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "exec"),
		log.String(common.FieldOperation, "query"),
		log.String(common.FieldStatus, "success"),
		log.Int(common.FieldCount, count),
		log.Int64(common.FieldDuration, duration))

	// 使用紧凑输出格式
	result := s.formatQueryCompact(columns, results, count, sqlQuery)

	return mcp.NewToolResultText(result), nil
}

// executeModification 执行修改操作（INSERT/UPDATE/DELETE等）
func (s *MySQLServer) executeModification(ctx context.Context, db *sql.DB, sqlQuery string) (*mcp.CallToolResult, error) {
	log.Info(ctx, "开始执行SQL修改操作",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "exec"),
		log.String(common.FieldOperation, "modify"),
		log.String(common.FieldSQL, sqlQuery))
	
	result, err := db.ExecContext(ctx, sqlQuery)
	if err != nil {
		log.Error(ctx, "执行失败",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldTool, "exec"),
			log.String(common.FieldOperation, "modify"),
			log.String(common.FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("执行失败: %v", err)), nil
	}
	
	// 获取影响的行数
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Warn(ctx, "无法获取影响行数",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "get_affected_rows"),
			log.String(common.FieldError, err.Error()))
		rowsAffected = -1
	}
	
	// 获取最后插入的ID（仅对INSERT有效）
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		log.Debug(ctx, "无法获取最后插入ID",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "get_last_insert_id"),
			log.String(common.FieldError, err.Error()))
		lastInsertId = -1
	}
	
	log.Info(ctx, "SQL修改操作执行成功", 
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "exec"),
		log.String(common.FieldOperation, "modify"),
		log.String(common.FieldStatus, "success"),
		log.Int64(common.FieldAffectedRows, rowsAffected),
		log.Int64("last_insert_id", lastInsertId))

	// 使用紧凑输出格式
	resultStr := s.formatModificationCompact(sqlQuery, rowsAffected, lastInsertId)

	return mcp.NewToolResultText(resultStr), nil
}

// formatQueryCompact 格式化查询结果为紧凑格式
func (s *MySQLServer) formatQueryCompact(columns []string, results []map[string]interface{}, count int, sqlQuery string) string {
	var result strings.Builder
	
	// 简化的SQL显示
	simplifiedSQL := sqlQuery
	if len(simplifiedSQL) > 50 {
		simplifiedSQL = simplifiedSQL[:47] + "..."
	}
	
	result.WriteString(fmt.Sprintf("Query: %s\n", simplifiedSQL))
	result.WriteString(fmt.Sprintf("Result: %d rows x %d columns\n\n", count, len(columns)))
	
	if count == 0 {
		result.WriteString("No data\n")
		return result.String()
	}
	
	// 如果结果太多，只显示前几行
	displayCount := count
	if displayCount > 10 {
		displayCount = 10
	}
	
	// 表格形式显示
	if len(columns) <= 5 { // 列数较少时使用表格
		// 计算列宽
		colWidths := make([]int, len(columns))
		for i, col := range columns {
			colWidths[i] = len(col)
		}
		
		for i := 0; i < displayCount; i++ {
			for j, col := range columns {
				val := fmt.Sprintf("%v", results[i][col])
				if len(val) > colWidths[j] {
					if len(val) > 20 {
						colWidths[j] = 20
					} else {
						colWidths[j] = len(val)
					}
				}
			}
		}
		
		// 表头
		for i, col := range columns {
			result.WriteString(fmt.Sprintf("%-*s", colWidths[i]+2, col))
		}
		result.WriteString("\n")
		
		// 分隔线
		for i := range columns {
			result.WriteString(strings.Repeat("-", colWidths[i]+2))
		}
		result.WriteString("\n")
		
		// 数据行
		for i := 0; i < displayCount; i++ {
			for j, col := range columns {
				val := fmt.Sprintf("%v", results[i][col])
				if len(val) > 20 {
					val = val[:17] + "..."
				}
				result.WriteString(fmt.Sprintf("%-*s", colWidths[j]+2, val))
			}
			result.WriteString("\n")
		}
	} else {
		// 列数较多时使用键值对形式
		for i := 0; i < displayCount; i++ {
			result.WriteString(fmt.Sprintf("--- Row %d ---\n", i+1))
			for _, col := range columns {
				val := fmt.Sprintf("%v", results[i][col])
				if len(val) > 50 {
					val = val[:47] + "..."
				}
				result.WriteString(fmt.Sprintf("%s: %s\n", col, val))
			}
			result.WriteString("\n")
		}
	}
	
	if count > displayCount {
		result.WriteString(fmt.Sprintf("... %d more rows not shown\n", count-displayCount))
	}
	
	return result.String()
}

// formatModificationCompact 格式化修改操作结果为紧凑格式
func (s *MySQLServer) formatModificationCompact(sqlQuery string, rowsAffected, lastInsertId int64) string {
	var result strings.Builder
	
	// 简化的SQL显示
	simplifiedSQL := sqlQuery
	if len(simplifiedSQL) > 50 {
		simplifiedSQL = simplifiedSQL[:47] + "..."
	}
	
	result.WriteString(fmt.Sprintf("Execute: %s\n", simplifiedSQL))
	
	// 根据SQL类型显示不同信息
	lowerSQL := strings.ToLower(strings.TrimSpace(sqlQuery))
	if strings.HasPrefix(lowerSQL, "insert") {
		result.WriteString("✓ Insert successful")
		if rowsAffected > 0 {
			result.WriteString(fmt.Sprintf(", affected %d rows", rowsAffected))
		}
		if lastInsertId > 0 {
			result.WriteString(fmt.Sprintf(", new ID: %d", lastInsertId))
		}
	} else if strings.HasPrefix(lowerSQL, "update") {
		result.WriteString("✓ Update successful")
		if rowsAffected > 0 {
			result.WriteString(fmt.Sprintf(", affected %d rows", rowsAffected))
		}
	} else if strings.HasPrefix(lowerSQL, "delete") {
		result.WriteString("✓ Delete successful")
		if rowsAffected > 0 {
			result.WriteString(fmt.Sprintf(", deleted %d rows", rowsAffected))
		}
	} else {
		result.WriteString("✓ Execute successful")
		if rowsAffected > 0 {
			result.WriteString(fmt.Sprintf(", affected %d rows", rowsAffected))
		}
	}
	
	result.WriteString("\n")
	return result.String()
}

// formatShowTablesCompact 格式化表列表为紧凑格式（英文输出）
func (s *MySQLServer) formatShowTablesCompact(tables []map[string]interface{}, database string) string {
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("Database: %s (%d tables)\n\n", database, len(tables)))
	
	if len(tables) == 0 {
		result.WriteString("No tables\n")
		return result.String()
	}
	
	for i, table := range tables {
		name := table["name"].(string)
		
		engine := ""
		if e, ok := table["engine"]; ok && e != nil {
			engine = fmt.Sprintf(" (%s)", e.(string))
		}
		
		result.WriteString(fmt.Sprintf("%d. %s%s", i+1, name, engine))
		
		// 添加注释
		if comment, ok := table["comment"]; ok && comment.(string) != "" {
			result.WriteString(fmt.Sprintf(" - %s", comment))
		}
		
		result.WriteString("\n")
		
		// 统计信息
		var stats []string
		if rows, ok := table["rows_estimate"]; ok && rows != nil {
			stats = append(stats, fmt.Sprintf("rows≈%v", rows))
		}
		if dataSize, ok := table["data_size"]; ok && dataSize != nil {
			stats = append(stats, fmt.Sprintf("data %s", s.formatBytes(dataSize.(int64))))
		}
		if indexSize, ok := table["index_size"]; ok && indexSize != nil {
			stats = append(stats, fmt.Sprintf("index %s", s.formatBytes(indexSize.(int64))))
		}
		
		if len(stats) > 0 {
			result.WriteString(fmt.Sprintf("   Stats: %s\n", strings.Join(stats, ", ")))
		}
		
		result.WriteString("\n")
	}
	
	return result.String()
}

// formatDescribeTableCompact 格式化单表结构为紧凑格式（英文输出，包含注释）
func (s *MySQLServer) formatDescribeTableCompact(tableName string, columns []map[string]interface{}) string {
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("Table: %s (%d columns)\n\n", tableName, len(columns)))
	
	if len(columns) == 0 {
		result.WriteString("No columns\n")
		return result.String()
	}
	
	// 表格形式显示列信息（包含注释）
	result.WriteString("Column            Type                 Constraints         Comment\n")
	result.WriteString("--------------------------------------------------------------------------------\n")
	
	for _, col := range columns {
		field := col["field"].(string)
		typ := col["type"].(string)
		comment := col["comment"].(string)
		
		// 构建约束信息
		var constraints []string
		
		if key := col["key"].(string); key != "" {
			switch key {
			case "PRI":
				constraints = append(constraints, "PK")
			case "UNI":
				constraints = append(constraints, "UNIQUE")
			case "MUL":
				constraints = append(constraints, "INDEX")
			}
		}
		
		if !col["null"].(bool) {
			constraints = append(constraints, "NOT NULL")
		}
		
		if extra := col["extra"].(string); extra != "" {
			if strings.Contains(extra, "auto_increment") {
				constraints = append(constraints, "AUTO_INC")
			}
		}
		
		constraintStr := strings.Join(constraints, ", ")
		if constraintStr == "" {
			constraintStr = "-"
		}
		
		// 截断过长的注释
		if len(comment) > 30 {
			comment = comment[:27] + "..."
		}
		if comment == "" {
			comment = "-"
		}
		
		result.WriteString(fmt.Sprintf("%-15s   %-20s %-18s %s\n", field, typ, constraintStr, comment))
	}
	
	result.WriteString("\n")
	return result.String()
}


// handleUpdateConfig 处理更新配置的请求
func (s *MySQLServer) handleUpdateConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理update_config请求",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "update_config"),
		log.String(common.FieldOperation, "update_config"))

	name := request.GetString("name", "")
	if name == "" {
		// 使用当前激活的数据库
		name = viper.GetString("active_database")
		if name == "" {
			name = "default"
		}
	}

	// 检查数据库配置是否存在
	dbKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(dbKey) {
		return mcp.NewToolResultError(fmt.Sprintf("Error: Database configuration '%s' does not exist", name)), nil
	}

	// 收集需要更新的配置
	updates := make(map[string]interface{})
	var updatedFields []string

	if host := request.GetString("host", ""); host != "" {
		updates[dbKey+".host"] = host
		updatedFields = append(updatedFields, "host")
	}

	if port := request.GetInt("port", 0); port > 0 {
		updates[dbKey+".port"] = port
		updatedFields = append(updatedFields, "port")
	}

	if user := request.GetString("user", ""); user != "" {
		updates[dbKey+".user"] = user
		updatedFields = append(updatedFields, "user")
	}

	if password := request.GetString("password", ""); password != "" {
		updates[dbKey+".password"] = password
		updatedFields = append(updatedFields, "password")
	}

	if database := request.GetString("database", ""); database != "" {
		updates[dbKey+".database"] = database
		updatedFields = append(updatedFields, "database")
	}

	if charset := request.GetString("charset", ""); charset != "" {
		updates[dbKey+".charset"] = charset
		updatedFields = append(updatedFields, "charset")
	}

	if len(updates) == 0 {
		return mcp.NewToolResultError("Error: No configuration properties to update"), nil
	}

	// 应用更新
	for key, value := range updates {
		viper.Set(key, value)
	}

	// 保存配置到文件
	if err := viper.WriteConfig(); err != nil {
		log.Error(ctx, "保存配置失败",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "save_config"),
			log.String(common.FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error saving configuration: %s", err.Error())), nil
	}

	// 清除连接池中的连接，强制重新连接
	if s.dbPool != nil {
		s.dbPool.CloseConnection(name)
		log.Info(ctx, "清除数据库连接池",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "clear_pool"),
			log.String(common.FieldDatabase, name))
	}

	// 格式化输出
	output := fmt.Sprintf("✅ Configuration updated for MySQL database '%s'\n", name)
	output += fmt.Sprintf("Updated fields (%d): %s\n", len(updatedFields), strings.Join(updatedFields, ", "))
	
	// 显示更新后的关键配置
	output += "\nCurrent configuration:\n"
	if host := viper.GetString(dbKey + ".host"); host != "" {
		output += fmt.Sprintf("  Host: %s\n", host)
	}
	if port := viper.GetInt(dbKey + ".port"); port > 0 {
		output += fmt.Sprintf("  Port: %d\n", port)
	}
	if user := viper.GetString(dbKey + ".user"); user != "" {
		output += fmt.Sprintf("  User: %s\n", user)
	}
	if database := viper.GetString(dbKey + ".database"); database != "" {
		output += fmt.Sprintf("  Database: %s\n", database)
	}
	if charset := viper.GetString(dbKey + ".charset"); charset != "" {
		output += fmt.Sprintf("  Charset: %s\n", charset)
	}

	log.Info(ctx, "配置更新成功", 
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "update_config"),
		log.String(common.FieldOperation, "update_config"),
		log.String(common.FieldStatus, "success"),
		log.String(common.FieldDatabase, name),
		log.String("fields", strings.Join(updatedFields, ",")))

	return mcp.NewToolResultText(output[:len(output)-1]), nil // 去掉最后的换行符
}

// handleGetConfigDetails 处理获取配置详情的请求
func (s *MySQLServer) handleGetConfigDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理get_config_details请求",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "get_config_details"),
		log.String(common.FieldOperation, "get_config"))

	name := request.GetString("name", "")
	includeSensitive := request.GetBool("include_sensitive", false)

	databases := viper.GetStringMap("databases")
	if len(databases) == 0 {
		return mcp.NewToolResultText("No MySQL databases configured"), nil
	}

	activeDatabase := viper.GetString("active_database")
	if activeDatabase == "" {
		activeDatabase = "default"
	}

	var output strings.Builder

	if name == "" {
		name = activeDatabase
	}

	if name == "all" {
		// 显示所有数据库的配置
		output.WriteString("📋 All MySQL Database Configurations\n")
		output.WriteString("====================================\n\n")

		for dbName, _ := range databases {
			output.WriteString(s.formatDatabaseConfig(dbName, dbName == activeDatabase, includeSensitive))
			output.WriteString("\n")
		}
	} else {
		// 显示指定数据库的配置
		if _, exists := databases[name]; !exists {
			return mcp.NewToolResultError(fmt.Sprintf("Error: Database configuration '%s' does not exist", name)), nil
		}
		
		output.WriteString(fmt.Sprintf("📋 MySQL Database Configuration: %s\n", name))
		output.WriteString("=====================================\n\n")
		output.WriteString(s.formatDatabaseConfig(name, name == activeDatabase, includeSensitive))
	}

	return mcp.NewToolResultText(output.String()), nil
}

// formatDatabaseConfig 格式化数据库配置信息
func (s *MySQLServer) formatDatabaseConfig(name string, isActive bool, includeSensitive bool) string {
	dbKey := fmt.Sprintf("databases.%s", name)
	
	var output strings.Builder
	
	// 数据库名称和状态
	status := "inactive"
	if isActive {
		status = "🟢 ACTIVE"
	} else {
		status = "⚪ inactive"
	}
	output.WriteString(fmt.Sprintf("Database: %s (%s)\n", name, status))
	
	// 基本配置
	if host := viper.GetString(dbKey + ".host"); host != "" {
		output.WriteString(fmt.Sprintf("  Host: %s\n", host))
	}
	
	if port := viper.GetInt(dbKey + ".port"); port > 0 {
		output.WriteString(fmt.Sprintf("  Port: %d\n", port))
	}
	
	if user := viper.GetString(dbKey + ".user"); user != "" {
		output.WriteString(fmt.Sprintf("  User: %s\n", user))
	}
	
	if password := viper.GetString(dbKey + ".password"); password != "" {
		if includeSensitive {
			output.WriteString(fmt.Sprintf("  Password: %s\n", password))
		} else {
			output.WriteString("  Password: *** (hidden)\n")
		}
	}
	
	if database := viper.GetString(dbKey + ".database"); database != "" {
		output.WriteString(fmt.Sprintf("  Database: %s\n", database))
	}
	
	if charset := viper.GetString(dbKey + ".charset"); charset != "" {
		output.WriteString(fmt.Sprintf("  Charset: %s\n", charset))
	}
	
	// 连接状态检查（仅对激活的数据库）
	if isActive {
		if db, err := s.getConnection(context.Background()); err == nil && db != nil {
			output.WriteString("  Connection: ✅ Available\n")
			// 尝试获取版本信息
			var version string
			if err := db.QueryRow("SELECT VERSION()").Scan(&version); err == nil {
				output.WriteString(fmt.Sprintf("  Server Version: %s\n", version))
			}
		} else {
			output.WriteString(fmt.Sprintf("  Connection: ❌ Failed (%s)\n", err.Error()))
		}
	}
	
	return output.String()
}

// validateSQLSecurity 使用SQL解析器验证SQL安全性
func (s *MySQLServer) validateSQLSecurity(ctx context.Context, sqlQuery string) error {
	// 解析SQL语句
	stmt, err := sqlparser.Parse(sqlQuery)
	if err != nil {
		log.Error(ctx, "SQL解析失败",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "parse_sql"),
			log.String(common.FieldSQL, sqlQuery),
			log.String(common.FieldError, err.Error()))
		return errors.Wrap(err, "SQL语句解析失败")
	}

	log.Debug(ctx, "SQL解析成功",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldOperation, "parse_sql"),
		log.String("statement_type", fmt.Sprintf("%T", stmt)))

	// 根据语句类型检查权限
	if err := s.validateStatementSecurity(ctx, stmt); err != nil {
		return err
	}
	
	return nil
}

// validateStatementSecurity 验证单个语句的安全性
func (s *MySQLServer) validateStatementSecurity(ctx context.Context, stmt sqlparser.Statement) error {
	switch stmt.(type) {
	// DDL 操作 - CREATE
	case *sqlparser.DDL:
		ddl := stmt.(*sqlparser.DDL)
		switch ddl.Action {
		case sqlparser.CreateStr:
			if s.disableCreate {
				return errors.New("CREATE操作已被禁用")
			}
		case sqlparser.DropStr:
			if s.disableDrop {
				return errors.New("DROP操作已被禁用")
			}
		case sqlparser.AlterStr:
			if s.disableAlter {
				return errors.New("ALTER操作已被禁用")
			}
		case sqlparser.TruncateStr:
			if s.disableTruncate {
				return errors.New("TRUNCATE操作已被禁用")
			}
		}
	// DML 操作
	case *sqlparser.Insert:
		// INSERT是写操作，不需要额外检查
		return nil
	case *sqlparser.Update:
		if s.disableUpdate {
			return errors.New("UPDATE操作已被禁用")
		}
	case *sqlparser.Delete:
		if s.disableDelete {
			return errors.New("DELETE操作已被禁用")
		}
	// 查询操作
	case *sqlparser.Select, *sqlparser.Show:
		// 查询操作，无需权限检查
		return nil
	default:
		// 对于其他类型的语句，记录警告但允许执行
		log.Warn(ctx, "未知的SQL语句类型",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "validate_security"),
			log.String("statement_type", fmt.Sprintf("%T", stmt)),
		)
		return nil
	}
	return nil
}

// isQueryStatement 判断是否为查询语句
func (s *MySQLServer) isQueryStatement(ctx context.Context, sqlQuery string) bool {
	// 解析SQL语句
	stmt, err := sqlparser.Parse(sqlQuery)
	if err != nil {
		log.Warn(ctx, "无法解析SQL判断查询类型，使用字符串前缀匹配",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "is_query"),
			log.String(common.FieldSQL, sqlQuery),
			log.String(common.FieldError, err.Error()))
		
		// 如果解析失败，回退到字符串匹配
		lowerSQL := strings.ToLower(strings.TrimSpace(sqlQuery))
		return strings.HasPrefix(lowerSQL, "select") || 
			strings.HasPrefix(lowerSQL, "show") || 
			strings.HasPrefix(lowerSQL, "describe") ||
			strings.HasPrefix(lowerSQL, "desc") ||
			strings.HasPrefix(lowerSQL, "explain")
	}

	// 基于AST类型判断
	switch stmt.(type) {
	case *sqlparser.Select, *sqlparser.Show:
		return true
	default:
		return false
	}
}
