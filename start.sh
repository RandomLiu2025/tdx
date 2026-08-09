#!/bin/sh

set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
STARTUP_TIMEOUT=${STARTUP_TIMEOUT:-120}
MODE=${1:-docker}

fail() {
    printf '错误: %s\n' "$1" >&2
    exit 1
}

show_logs() {
    printf '%s\n' '最近的服务日志:' >&2
    docker compose logs --tail=100 tdx-api >&2 || true
}

print_usage() {
    printf '%s\n' \
        '用法: ./start.sh [docker|local]' \
        '' \
        '  docker  使用 Docker Compose 构建并启动服务（默认）' \
        '  local   使用本机 Go 工具链构建并前台运行服务'
}

start_docker() {
    case "$STARTUP_TIMEOUT" in
        ''|*[!0-9]*) fail "STARTUP_TIMEOUT 必须是正整数秒数" ;;
    esac
    [ "$STARTUP_TIMEOUT" -gt 0 ] || fail "STARTUP_TIMEOUT 必须大于 0"

    command -v docker >/dev/null 2>&1 || fail "未安装 Docker，请先安装并启动 Docker"
    docker info >/dev/null 2>&1 || fail "Docker daemon 未运行，请先启动 Docker"
    docker compose version >/dev/null 2>&1 || fail "当前 Docker 未安装 Compose 插件"

    cd "$ROOT_DIR"

    printf '%s\n' '正在构建并启动 tdx-api...'
    if ! docker compose up --detach --build; then
        fail "Docker Compose 启动失败"
    fi

    container_id=$(docker compose ps --quiet tdx-api)
    if [ -z "$container_id" ]; then
        show_logs
        fail "未找到 tdx-api 容器"
    fi

    deadline=$(( $(date +%s) + STARTUP_TIMEOUT ))
    printf '等待健康检查通过，超时时间 %s 秒...\n' "$STARTUP_TIMEOUT"

    while [ "$(date +%s)" -lt "$deadline" ]; do
        container_status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container_id" 2>/dev/null || printf 'missing')

        case "$container_status" in
            healthy|running)
                published_address=$(docker compose port tdx-api 8080 2>/dev/null | sed -n '1p')
                published_port=${published_address##*:}
                if [ -z "$published_port" ]; then
                    published_port=${TDX_API_PORT:-8080}
                fi
                printf 'tdx-api 启动成功: http://localhost:%s/\n' "$published_port"
                return 0
                ;;
            unhealthy|exited|dead|missing)
                show_logs
                fail "tdx-api 容器状态异常: $container_status"
                ;;
        esac

        sleep 2
    done

    show_logs
    fail "等待 tdx-api 健康检查超时"
}

start_local() {
    command -v go >/dev/null 2>&1 || fail "未安装 Go，请先安装 Go 1.23 或更高版本"

    cd "$ROOT_DIR"
    binary_dir="$ROOT_DIR/.local/bin"
    binary_path="$binary_dir/tdx-api"
    local_goproxy=${GOPROXY:-https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct}

    mkdir -p "$binary_dir"
    printf '%s\n' '正在使用本机 Go 工具链构建 tdx-api...'
    if ! GOPROXY="$local_goproxy" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$binary_path" ./cmd/tdx-api; then
        fail "本地构建失败"
    fi

    printf '本地服务即将启动，监听地址 %s；按 Ctrl+C 停止。\n' "${TDX_HTTP_ADDR:-:8080}"
    exec "$binary_path"
}

if [ "$#" -gt 1 ]; then
    print_usage >&2
    exit 1
fi

case "$MODE" in
    docker) start_docker ;;
    local) start_local ;;
    -h|--help|help) print_usage ;;
    *)
        print_usage >&2
        fail "不支持的启动模式: $MODE"
        ;;
esac
