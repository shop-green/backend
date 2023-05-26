set -euxo pipefail

echo "$(pwd)"
ls -la
mkdir -p "$(pwd)/functions"
cd src
GOBIN=$(pwd)/../functions go install ./...
cd ..
chmod +x "$(pwd)"/functions/*
go env
