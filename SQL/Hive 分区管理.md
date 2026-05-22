# 📝 Hive 分区管理笔记：获取最新分区并解析参数

> **适用场景**：自动化调度任务、增量数据同步、动态分区查询  
> **核心目标**：编程式获取表最新分区 → 提取 `dt`/`hh` 参数 → 用于下游任务

---

## 🔑 核心逻辑流程图

```
┌─────────────────────────┐
│ 1. SHOW PARTITIONS 表名  │
└────────┬────────────────┘
         ▼
┌─────────────────────────┐
│ 2. 按分区名倒序排序      │
│    (字符串字典序 = 时间序)│
└────────┬────────────────┘
         ▼
┌─────────────────────────┐
│ 3. LIMIT 1 取最新分区   │
│    结果: dt=20260522/hour=14 │
└────────┬────────────────┘
         ▼
┌─────────────────────────┐
│ 4. 字符串解析提取参数    │
│    → dt="20260522", hh="14" │
└─────────────────────────┘
```

---

## 💻 标准代码模板（PySpark）

```python
from pyspark.sql import SparkSession

spark_session = SparkSession.builder.getOrCreate()

# ① 获取分区列表
partitions_df = spark_session.sql("SHOW PARTITIONS ads.ads_device_ta_merge_hourly")

# ② 按分区路径倒序排序，取最新1个（字典序 ≈ 时间序，需分区格式规范）
latest_partition = partitions_df.orderBy("partition", ascending=False).limit(1)

# ③ 收集结果并提取路径字符串
partition_path = latest_partition.collect()[0][0]  # 例: "dt=20260522/hour=14"

# ④ 解析分区参数（通用解析函数）
def parse_partition_path(partition_str):
    """
    解析 Hive 分区路径，返回 dict
    输入: "dt=20260522/hour=14" 或 "dt=20260522/hour=14/region=cn"
    输出: {"dt": "20260522", "hour": "14", "region": "cn"}
    """
    result = {}
    for part in partition_str.split("/"):
        if "=" in part:
            key, value = part.split("=", 1)  # split 1 次，避免 value 含 =
            result[key] = value
    return result

# ⑤ 使用解析结果
params = parse_partition_path(partition_path)
dt_last = params.get("dt")
hh_last = params.get("hour")  # 注意：字段名是 hour 而非 hh

print(f"最新分区: dt={dt_last}, hour={hh_last}")
```

---

## 🧩 参数解析方案对比

| 方案 | 代码示例 | 优点 | 缺点 | 推荐场景 |
|------|----------|------|------|----------|
| ✅ **通用循环解析**（本例） | `for part in path.split("/")` | 灵活、支持任意分区键、易维护 | 代码略长 | 多分区键、动态解析 |
| **字符串切片** | `path.split("dt=")[1].split("/")[0]` | 简洁 | 脆弱、分区顺序变化即失效 | 固定单分区键、快速脚本 |
| **正则表达式** | `re.search(r"dt=(\d+)", path).group(1)` | 精确匹配、容错强 | 需 import re、可读性略低 | 复杂分区格式、生产环境 |
| **Spark SQL 解析** | `spark.sql(f"SELECT * FROM table WHERE dt='{dt}'")` | 无需手动解析 | 需已知参数值 | 已知分区参数直接查询 |

> 💡 **建议**：封装 `parse_partition_path()` 函数复用，避免重复造轮子。

---

## ⚠️ 关键注意事项

### 1️⃣ 分区命名规范决定排序正确性
```sql
-- ✅ 推荐：零填充 + 字典序 = 时间序
dt=20260522/hour=09   -- 字符串排序正确
dt=20260522/hour=14

-- ❌ 避免：无前导零，字典序 ≠ 时间序
dt=2026-5-22/hour=9   -- "hour=9" > "hour=14" (字符串比较)
```

### 2️⃣ 空表/无分区处理
```python
if partitions_df.isEmpty():
    raise ValueError("表无分区或不存在")
    
latest = partitions_df.orderBy("partition", ascending=False).limit(1)
if latest.count() == 0:
    # 降级方案：使用默认分区或报错
    dt_last = "19700101"
```

### 3️⃣ 性能优化建议
| 场景 | 优化方案 |
|------|----------|
| 分区数 > 1000 | `SHOW PARTITIONS` 可能慢，改用 `spark.catalog.listPartitions("db.table")`（Spark 2.3+） |
| 高频调用 | 缓存分区列表：`spark_session.sparkContext.broadcast(partitions_list)` |
| 跨集群同步 | 优先使用 `msck repair table` 确保元数据最新 |

### 4️⃣ 字段名一致性
```python
# SHOW PARTITIONS 返回列名固定为 "partition"（字符串）
# 解析时注意分区键名：hour ≠ hh, ds ≠ dt
# 建议统一团队分区命名规范：dt(日期), hh/hour(小时), region(地域)
```

---

## 🔄 进阶：替代方案与扩展

### 方案1：使用 `spark.catalog.listPartitions`（更 Spark-native）
```python
# 返回 Partition 对象列表，无需字符串解析
partitions = spark_session.catalog.listPartitions("ads.ads_device_ta_merge_hourly")

# 按分区值排序（支持多分区键）
latest = sorted(
    partitions, 
    key=lambda p: (p.values.get("dt", ""), p.values.get("hour", "")), 
    reverse=True
)[0]

dt_last = latest.values.get("dt")
hh_last = latest.values.get("hour")
```
✅ 优势：类型安全、避免字符串解析错误  
⚠️ 注意：Spark 2.3+ 支持，部分旧版本不可用

### 方案2：直接查询最大分区值（精准但需已知分区键）
```sql
-- 单分区键
SELECT MAX(dt) AS latest_dt FROM ads.ads_device_ta_merge_hourly

-- 双分区键（先最大 dt，再最大 hour）
SELECT dt, MAX(hour) AS latest_hour 
FROM ads.ads_device_ta_merge_hourly 
WHERE dt = (SELECT MAX(dt) FROM ads.ads_device_ta_merge_hourly)
GROUP BY dt
```
✅ 优势：100% 准确，不依赖字符串排序  
⚠️ 注意：需扫描数据/元数据，大表可能慢

### 方案3：封装为工具函数（生产推荐）
```python
def get_latest_partition_params(spark, table_name, partition_keys=["dt", "hour"]):
    """
    获取 Hive 表最新分区参数
    :param spark: SparkSession
    :param table_name: "db.table"
    :param partition_keys: 分区键列表，按优先级排序
    :return: dict like {"dt": "20260522", "hour": "14"}
    """
    try:
        # 优先使用 catalog API
        partitions = spark.catalog.listPartitions(table_name)
        if not partitions:
            return None
            
        # 多键排序：先按 dt，再按 hour
        def sort_key(p):
            return tuple(p.values.get(k, "") for k in partition_keys)
            
        latest = sorted(partitions, key=sort_key, reverse=True)[0]
        return {k: latest.values.get(k) for k in partition_keys}
        
    except Exception as e:
        # 降级方案：SHOW PARTITIONS + 字符串解析
        df = spark.sql(f"SHOW PARTITIONS {table_name}")
        if df.isEmpty():
            return None
        path = df.orderBy("partition", ascending=False).limit(1).collect()[0][0]
        return parse_partition_path(path)  # 复用前述解析函数
```

---

## 🧪 调试与验证技巧

```python
# ① 预览分区列表（前10个）
partitions_df.show(10, truncate=False)

# ② 验证解析结果
print(f"原始路径: {partition_path}")
print(f"解析结果: {params}")

# ③ 用解析参数反向查询验证
query = f"SELECT COUNT(1) FROM ads.ads_device_ta_merge_hourly WHERE dt='{dt_last}' AND hour='{hh_last}'"
spark_session.sql(query).show()

# ④ 检查分区键顺序（影响排序）
spark_session.sql("DESCRIBE FORMATTED ads.ads_device_ta_merge_hourly") \
    .filter("col_name LIKE '%Partition Information%'").show(20, truncate=False)
```

---

## 📌 最佳实践清单

```markdown
- [ ] 分区字段使用零填充格式：dt=YYYYMMDD, hour=HH（2位）
- [ ] 封装解析函数，避免重复代码 + 统一错误处理
- [ ] 优先使用 `spark.catalog.listPartitions`（类型安全）
- [ ] 大表查询前加 `LIMIT` 或分区过滤，避免全表扫描
- [ ] 关键任务添加空值/异常校验，避免静默失败
- [ ] 记录日志：`logging.info(f"Using partition: {partition_path}")`
```

---

## 🔗 附：Hive/Spark 分区相关命令速查

```sql
-- 查看表分区结构
DESCRIBE FORMATTED db.table;

-- 列出所有分区
SHOW PARTITIONS db.table;
SHOW PARTITIONS db.table PARTITION(dt='20260522');  -- 过滤

-- 修复元数据（新增分区未同步时）
MSCK REPAIR TABLE db.table;

-- 动态分区插入（写入时自动创建分区）
SET hive.exec.dynamic.partition=true;
INSERT INTO TABLE db.table PARTITION(dt, hour)
SELECT ..., dt_col, hour_col FROM source;
```

---

> 💡 **一句话总结**：  
> `SHOW PARTITIONS + 倒序排序 + 字符串解析` 是获取 Hive 最新分区的通用方案；生产环境建议封装函数 + 添加降级逻辑 + 优先使用 `spark.catalog.listPartitions` 提升健壮性。

---
*最后更新：2026-05-22 | 适用 Spark 版本：2.4+ / 3.0+ | Hive 版本：2.3+*
