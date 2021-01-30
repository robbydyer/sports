#!/bin/bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"

cd "${ROOT}"

echo "---" > "${ROOT}/pkg/mlbmock/assets/mock_games.yaml"
echo "---" > "${ROOT}/pkg/mlbmock/assets/mock_livegames.yaml"
echo "---" > "${ROOT}/pkg/mlbmock/assets/mock_teams.yaml"

for i in `cat "${ROOT}/script/mlb_ids.txt"`; do
id=$(echo $i | cut -f1 -d,)
t=$(echo $i | cut -f2 -d,)
cat <<EOF >> "${ROOT}/pkg/mlbmock/assets/mock_teams.yaml"
- id: ${id}
  abbreviation: ${t}
  name: ${t}
EOF
cat <<EOF >> "${ROOT}/pkg/mlbmock/assets/mock_games.yaml"
- gamePk: ${id}
  link: "${id}"
  teams:
    away:
      team:
        id: ${id}
        abbreviation: "${t}"
        name: "${t}"
    home:
      team:
        id: ${id}
        abbreviation: "${t}"
        name: "${t}"
EOF
cat <<EOF >> "${ROOT}/pkg/mlbmock/assets/mock_livegames.yaml"
- gamePk: ${id}
  link: "${id}"
  gameData:
  liveData:
    linescore:
      currentInning 3
      currentInningOrdinal: "3rd"
      inningState: "Bottom"
      teams:
        runs 1
        away:
          team:
            id: ${id}
            abbreviation: "${t}"
            name: "${t}"
        home:
          runs 2
          team:
            id: ${id}
            abbreviation: "${t}"
            name: "${t}"
EOF
done
