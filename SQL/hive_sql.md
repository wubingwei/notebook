# Hive

> COALESCE(win_price, bid_price))
> 按优先级获取第一个有值的字段


> 获取hive表最新的分区
```SQL
partitions_df = spark_session.sql("SHOW PARTITIONS ads.ads_device_ta_merge_hourly")

# Extract the latest partitions based on dt and hh
latest_partitions = partitions_df.orderBy("partition", ascending=False).limit(1)

# Show the latest partitions
#latest_partitions.show(truncate=False)

latest_partition_path = latest_partitions.collect()[0][0]

# # 从路径中拆分出 dt 和 hh
partition_parts = latest_partition_path.split("/")
dt_last = None
hh_last = None
for part in partition_parts:
    if part.startswith("dt="):
        dt_last = part.split("=")[1]
    elif part.startswith("hour="):
        hh_last = part.split("=")[1]
```
