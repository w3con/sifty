#!/bin/bash
# usage: fetch.sh  (reads list from stdin: key<TAB>url). 4 s pause between requests, project UA, no retries.
UA='Sifty/1.0 (personal read-only job discovery)'
first=1
while IFS=$'\t' read -r key url; do
  [ -z "$key" ] && continue
  [ $first -eq 0 ] && sleep 4
  first=0
  res=$(curl -sS -L --max-time 30 -A "$UA" -H 'Accept: text/html,application/json,application/rss+xml,*/*' -o "pages/$key.body" -w '%{http_code} %{size_download} %{content_type} %{url_effective}' "$url" 2>&1)
  echo "$key	$url	$res"
done
