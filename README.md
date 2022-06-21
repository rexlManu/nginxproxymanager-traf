# nginxproxymanager-traf

Log your access traffic to influxdb.

## Setup

```bash
docker build -t nginxproxymanager-traf .

docker run -d --name nginxproxymanager-traf \
    -e INFLUXDB_URL=influxdb \
    -e INFLUXDB_TOKEN=influxdb \
    -e INFLUXDB_BUCKET=influxdb \
    -e INFLUXDB_ORG=influxdb \
    -v nginx-logs:/logs \
    -v GeoLite2-City.mmdb:/app/GeoLite2-City.mmdb \
    nginxproxymanager-traf
```