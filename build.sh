set -euxo pipefail

mkdir -p "$(pwd)/functions"
cd src
GOBIN=$(pwd)/../functions go install ./...
cd ..
chmod +x "$(pwd)"/functions/*
# go env
