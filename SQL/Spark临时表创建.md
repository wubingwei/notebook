# 📝 Spark 建表方法笔记：DataFrame API + 临时视图注册

> **适用场景**：Zeppelin / Jupyter Notebook 交互式分析、临时数据探查、SQL/PySpark 混合开发  
> **核心术语**：`Schema-on-Read` · `Temporary View` · `DataFrame API`

---

## 🔖 核心概念

```python
# 标准代码模板
df = spark.read \
    .option("delimiter", "\t") \
    .option("nullValue", "") \
    .option("header", "false") \
    .schema("field1 STRING, field2 INT, ...") \  # ✅ 显式 Schema
    .csv("s3a://bucket/path/")                    # ✅ Schema-on-Read

df.createOrReplaceTempView("table_name")          # ✅ 注册临时视图
```

| 术语 | 含义 | 本例体现 |
|------|------|----------|
| **Schema-on-Read** | 读取数据时定义结构，而非写入时 | `.schema("...")` 显式声明字段类型 |
| **DataFrame API** | Spark 编程式数据操作接口 | `spark.read...csv()` 链式调用 |
| **临时视图（TempView）** | 会话级虚拟表，仅映射 DataFrame，不存储数据 | `createOrReplaceTempView("table_name")` |
| **无持久化** | 视图随 SparkSession 结束而消失，不写元数据到 Hive | 适合 Notebook 临时分析 |

---

## 🆚 Spark「建表」方式对比

| 方式 | 代码示例 | 持久化 | 生命周期 | 适用场景 |
|------|----------|--------|----------|----------|
| ✅ **临时视图**（本例） | `df.createOrReplaceTempView("t")` | ❌ | 当前 SparkSession | 交互式分析、临时 SQL 查询 |
| **全局临时视图** | `df.createOrReplaceGlobalTempView("t")` | ❌ | 应用级（跨 Session） | 多 Notebook 共享临时表 |
| **Hive 托管表** | `df.write.saveAsTable("db.tbl")` | ✅ | 永久 | 生产复用、权限管控、元数据管理 |
| **Spark SQL DDL** | `CREATE TABLE t USING CSV OPTIONS(...) LOCATION '...'` | ✅/❌ | 可选 | 声明式建表、运维脚本 |
| **外部表（External）** | `CREATE EXTERNAL TABLE ... LOCATION 's3a://...'` | ✅（仅元数据） | 永久 | 数据湖架构、多引擎共享数据 |

> 💡 **记忆口诀**：  
> `临时视图` = 内存别名｜`saveAsTable` = 入库持久化｜`EXTERNAL` = 元数据托管 + 数据外置

---

## ⚙️ 关键参数说明

### 📥 读取配置（`spark.read`）
```python
.option("delimiter", "\t")      # 分隔符：\t=TSV, ,=CSV
.option("nullValue", "")        # 空字符串识别为 NULL
.option("header", "false")      # 首行是否为字段名
.schema("name STRING, age INT") # 显式 Schema（推荐生产使用）
```

### 🗂️ Schema 定义格式
```python
# 字符串格式（本例）
"field1 STRING, field2 INT, field3 DOUBLE, field4 TIMESTAMP"

# 等价 StructType 格式（更灵活，支持嵌套）
from pyspark.sql.types import *
schema = StructType([
    StructField("field1", StringType(), True),
    StructField("field2", IntegerType(), True),
    # ...
])
```

### 🪄 视图注册
```python
# 临时视图（推荐）
df.createOrReplaceTempView("view_name")  # 会话级，可覆盖

# 全局临时视图（需加前缀）
df.createOrReplaceGlobalTempView("view_name")
# 查询时需：SELECT * FROM global_temp.view_name
```

---

## ✅ 优势与注意事项

### 🌟 优势
- **灵活**：无需预建表，读时定义 Schema，适合探索性分析
- **轻量**：不写元数据，不产生存储开销
- **兼容**：注册后可直接用 `%sql` 写标准 SQL，降低 PySpark/SQL 切换成本
- **调试友好**：`df.show()` / `df.printSchema()` 快速验证数据

### ⚠️ 注意事项
| 问题 | 建议方案 |
|------|----------|
| **视图生命周期短** | 仅当前 Session 有效，重启内核后消失；需持久化请用 `saveAsTable` |
| **字段名大小写** | Spark 默认区分大小写，建议 SQL 中用 `` `backtick` `` 包裹字段 |
| **S3 权限配置** | 确保 EMR/Spark 角色有 `s3a://` 读取权限，必要时配置 `fs.s3a.access.key` |
| **Schema 维护成本** | 显式 Schema 需人工维护，源数据变更易报错；可临时用 `inferSchema=True` 推断（生产慎用） |
| **性能** | 临时视图不缓存数据，重复查询建议 `df.cache()` 或 `spark.catalog.cacheTable()` |

---

## 🛠️ 实用技巧扩展

### 🔁 封装读取逻辑（复用）
```python
def load_tsv_table(spark, path, schema_str, view_name):
    df = spark.read \
        .option("delimiter", "\t") \
        .option("nullValue", "") \
        .schema(schema_str) \
        .csv(path)
    df.createOrReplaceTempView(view_name)
    return df

# 使用
schema = "mmp_name STRING, demand_package_name STRING, exp_id INT, ..."
load_tsv_table(spark, "s3a://.../ios_joined/", schema, "ab_ios1_adjust")
```

### 🧪 调试与验证
```python
df.printSchema()                          # 树状展示字段类型
df.select("demand_package_name").distinct().show(10)  # 抽样验证
spark.sql("DESCRIBE ab_ios1_adjust").show()  # 通过 SQL 查看视图结构
```

### 🔄 升级为持久化表（生产推荐）
```python
# 方式1：DataFrame API
df.write.mode("overwrite").saveAsTable("dwd.ios_joined_daily")

# 方式2：Spark SQL（支持分区）
spark.sql("""
  CREATE TABLE IF NOT EXISTS dwd.ios_joined_daily
  USING PARQUET
  PARTITIONED BY (dt STRING)
  LOCATION 's3a://mob-emr-test/warehouse/dwd/ios_joined_daily/'
""")
df.write.mode("overwrite").insertInto("dwd.ios_joined_daily", overwritePartitions=True)
```

### 🔍 查询示例（注册后）
```sql
-- %sql 单元格中直接执行
SELECT 
  platform,
  COUNT(DISTINCT demand_package_name) AS pkg_cnt,
  AVG(date_diff) AS avg_diff
FROM ab_ios1_adjust
WHERE exploded_ins_date >= '2026-05-01'
GROUP BY platform
ORDER BY pkg_cnt DESC;
```

---

## 📌 最佳实践清单

```markdown
- [ ] Schema 显式定义，避免 inferSchema 导致类型错误
- [ ] 敏感路径/配置使用变量或配置中心管理，避免硬编码
- [ ] 临时视图命名加前缀（如 `tmp_` / `view_`），便于识别
- [ ] 大表查询前加 `LIMIT` 或分区过滤，避免全表扫描
- [ ] 重要中间结果及时 `cache()` 或持久化，避免重复计算
- [ ] Notebook 结束前 `spark.catalog.dropTempView("view_name")` 清理资源
```

---

## 🔗 相关文档
- [Spark SQL, DataFrames and Datasets Guide](https://spark.apache.org/docs/latest/sql-programming-guide.html)
- [Create Temporary Views in Spark](https://spark.apache.org/docs/latest/api/python/reference/pyspark.sql/api/pyspark.sql.DataFrame.createOrReplaceTempView.html)
- [Spark on S3 Best Practices](https://docs.aws.amazon.com/emr/latest/ReleaseGuide/emr-spark-s3.html)

> 💡 **一句话总结**：  
> `spark.read + schema + createOrReplaceTempView` 是 Spark 交互式分析的「黄金组合」——灵活、轻量、兼容 SQL，适合快速验证；生产复用请升级为 `saveAsTable` 持久化表。

---
*最后更新：2026-05-22 | 适用 Spark 版本：2.4+ / 3.0+*
