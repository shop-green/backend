# Kinetica

## Create table

```sql
CREATE TABLE farmers (
    id VARCHAR(32) PRIMARY KEY,
    longitude FLOAT,
    latitude FLOAT
);
```

## Query table

```sql
SELECT id, GEODIST(farmers.longitude, farmers.latitude, -73.9, 40.6) AS distance_m
FROM farmers
WHERE GEODIST(farmers.longitude, farmers.latitude, -73.9, 40.6) < 30000;
```
