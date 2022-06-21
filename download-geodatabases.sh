#!/usr/bin/env bash
echo "Deleting old geodatabases..."
rm -f GeoLite2-*.mmdb
wget https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-ASN.mmdb -O GeoLite2-ASN.mmdb
wget https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb -O GeoLite2-ASN.mmdb
echo "Finish!"