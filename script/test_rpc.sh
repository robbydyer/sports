#!/bin/bash

set -ex
path="matrix.v1.Sportsmatrix"

curl \
  -X POST \
  --header "Content-Type: application/json" \
  -d '{}' \
  "http://localhost:8080/${path}/Status"
echo
exit

curl \
  -X POST \
  --header "Content-Type: application/json" \
  -d '{"enabled":false}' \
  "http://localhost:8080/${path}/SetAll"
echo
sleep 4
curl \
  -X POST \
  --header "Content-Type: application/json" \
  -d '{"board":"mlb"}' \
  "http://localhost:8080/${path}/Jump"
echo

curl \
  -X POST \
  --header "Content-Type: application/json" \
  -d '{}' \
  "http://localhost:8080/mlb/sport.v1.Sport/GetStatus"

echo
curl \
  -X POST \
  --header "Content-Type: application/json" \
  -d '{"enabled":false, "favorite_hidden":false, "favorite_sticky":false, "scroll_enabled":true, "tight_scroll_enabled":true, "record_rank_enabled":false, "odds_enabled":false}' \
  "http://localhost:8080/mlb/sport.v1.Sport/SetStatus"

echo
sleep 3
curl \
  -X POST \
  --header "Content-Type: application/json" \
  -d '{"enabled":true, "favorite_hidden":false, "favorite_sticky":false, "scroll_enabled":false, "tight_scroll_enabled":true, "record_rank_enabled":false, "odds_enabled":false}' \
  "http://localhost:8080/mlb/sport.v1.Sport/SetStatus"
echo "SHOULD BE ENABLED"
sleep 3
curl \
  -X POST \
  --header "Content-Type: application/json" \
  -d '{"enabled":true, "favorite_hidden":false, "favorite_sticky":false, "scroll_enabled":false, "tight_scroll_enabled":true, "record_rank_enabled":false, "odds_enabled":false}' \
  "http://localhost:8080/mlb/sport.v1.Sport/SetStatus"

echo
curl \
  -X POST \
  --header "Content-Type: application/json" \
  -d '{}' \
  "http://localhost:8080/mlb/sport.v1.Sport/GetStatus"
