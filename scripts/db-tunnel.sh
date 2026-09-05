#!/bin/bash

# Gestão de túnel SSH para a base de dados Postgres do caetano.
# Uso: ./scripts/db-tunnel.sh {start|stop|status|restart} [postgres] [tunnel-host] [local-port]
#
# tunnel-host é o alias SSH (definido em ~/.ssh/config do operador) por onde se
# chega à BD — dev e produção usam hosts diferentes (ver
# caetano.dev.yaml.example / caetano.prod.yaml.example). local-port permite
# correr o túnel de dev e o de produção ao mesmo tempo, em portas distintas
# (necessário para a migração pontual de RF-11).

ACTION="$1"
DB_TYPE="${2:-postgres}"
TUNNEL_HOST="${3:?Erro: indica o alias SSH da BD (ex.: postgres ou postgres-pgvector)}"
LOCAL_PORT="${4:-5432}"
REMOTE_HOST="localhost"

case "$DB_TYPE" in
    postgres)
        REMOTE_PORT="5432"
        TEST_CMD="pg_isready"
        TEST_ARGS="-h localhost -p $LOCAL_PORT -U postgres"
        ;;
    mysql)
        REMOTE_PORT="3306"
        TEST_CMD="mysqladmin"
        TEST_ARGS="ping -h localhost -P $LOCAL_PORT"
        ;;
    *)
        echo "Erro: tipo de BD inválido. Usa 'postgres' ou 'mysql'"
        exit 1
        ;;
esac

PID_FILE="/tmp/caetano-${DB_TYPE}-tunnel-${LOCAL_PORT}.pid"

test_port_connectivity() {
    local port=$1
    local timeout=${2:-2}
    if command -v nc >/dev/null 2>&1; then
        timeout "$timeout" nc -z localhost "$port" >/dev/null 2>&1
        return $?
    elif command -v timeout >/dev/null 2>&1; then
        timeout "$timeout" bash -c "</dev/tcp/localhost/$port" >/dev/null 2>&1
        return $?
    else
        (echo >/dev/tcp/localhost/"$port") >/dev/null 2>&1
        return $?
    fi
}

check_tunnel_health() {
    local pid=$1
    if ! ps -p "$pid" > /dev/null 2>&1; then
        return 1
    fi
    if ! lsof -i :"$LOCAL_PORT" -sTCP:LISTEN >/dev/null 2>&1; then
        return 1
    fi
    if ! test_port_connectivity "$LOCAL_PORT" 2; then
        return 1
    fi
    return 0
}

start_tunnel() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if [ -n "$PID" ] && ps -p "$PID" > /dev/null 2>&1; then
            if check_tunnel_health "$PID"; then
                echo "Túnel $DB_TYPE ($TUNNEL_HOST:$LOCAL_PORT) já ativo (PID: $PID)"
                return 0
            fi
            echo "Túnel encontrado mas inativo (PID: $PID), a limpar..."
            kill "$PID" 2>/dev/null
            rm "$PID_FILE"
            sleep 1
        else
            rm "$PID_FILE"
        fi
    fi

    if lsof -i :"$LOCAL_PORT" -sTCP:LISTEN >/dev/null 2>&1 || ss -tln | grep -q ":$LOCAL_PORT "; then
        EXISTING_PID=$(lsof -ti :"$LOCAL_PORT" -sTCP:LISTEN 2>/dev/null | head -n1)
        if [ -n "$EXISTING_PID" ] && ps -p "$EXISTING_PID" -o comm= 2>/dev/null | grep -q ssh; then
            if check_tunnel_health "$EXISTING_PID"; then
                echo "$EXISTING_PID" > "$PID_FILE"
                echo "Túnel SSH já existente na porta $LOCAL_PORT reaproveitado (PID: $EXISTING_PID)"
                return 0
            fi
            echo "Túnel SSH não funcional na porta $LOCAL_PORT (PID: $EXISTING_PID), a reiniciar..."
            kill "$EXISTING_PID" 2>/dev/null
            sleep 2
        fi
        if lsof -i :"$LOCAL_PORT" -sTCP:LISTEN >/dev/null 2>&1; then
            echo "Erro: a porta $LOCAL_PORT já está em uso por outro processo"
            return 1
        fi
    fi

    echo "A iniciar túnel SSH para $DB_TYPE via $TUNNEL_HOST (porta local $LOCAL_PORT)..."
    ssh -f -N -L "$LOCAL_PORT:$REMOTE_HOST:$REMOTE_PORT" "$TUNNEL_HOST"
    if [ $? -ne 0 ]; then
        echo "Falha ao iniciar túnel SSH"
        return 1
    fi

    sleep 2
    SSH_PID=$(lsof -ti :"$LOCAL_PORT" 2>/dev/null | head -n1)
    if [ -z "$SSH_PID" ]; then
        echo "Aviso: não foi possível determinar o PID do túnel"
        return 1
    fi

    echo "$SSH_PID" > "$PID_FILE"
    echo "Túnel $DB_TYPE iniciado com PID: $SSH_PID"

    if test_port_connectivity "$LOCAL_PORT" 3; then
        echo "✓ Porta $LOCAL_PORT a responder"
        if command -v "$TEST_CMD" >/dev/null 2>&1; then
            if $TEST_CMD $TEST_ARGS >/dev/null 2>&1; then
                echo "✓ BD $DB_TYPE acessível via túnel"
            else
                echo "⚠ Túnel ativo mas BD $DB_TYPE não responde"
            fi
        fi
    else
        echo "⚠ Aviso: túnel iniciado mas porta ainda não responde"
    fi
    return 0
}

stop_tunnel() {
    if [ ! -f "$PID_FILE" ]; then
        echo "Sem ficheiro PID para $DB_TYPE:$LOCAL_PORT (túnel não ativo)"
        return 0
    fi
    PID=$(cat "$PID_FILE")
    echo "A parar túnel $DB_TYPE:$LOCAL_PORT (PID: $PID)..."
    if kill "$PID" 2>/dev/null; then
        echo "Túnel parado"
    else
        echo "Falha ao parar (PID: $PID) — a remover ficheiro PID obsoleto"
    fi
    rm -f "$PID_FILE"
    return 0
}

status_tunnel() {
    if [ ! -f "$PID_FILE" ]; then
        echo "Túnel $DB_TYPE:$LOCAL_PORT não está ativo"
        return 1
    fi
    PID=$(cat "$PID_FILE")
    if ! ps -p "$PID" > /dev/null 2>&1; then
        echo "Ficheiro PID existe mas o processo não está ativo"
        rm -f "$PID_FILE"
        return 1
    fi
    echo "Túnel $DB_TYPE:$LOCAL_PORT em execução (PID: $PID)"
    if check_tunnel_health "$PID"; then
        echo "✓ Túnel saudável, porta $LOCAL_PORT a responder"
        if command -v "$TEST_CMD" >/dev/null 2>&1 && $TEST_CMD $TEST_ARGS >/dev/null 2>&1; then
            echo "✓ BD $DB_TYPE acessível"
        fi
    else
        echo "✗ Processo existe mas o túnel não está funcional"
        return 1
    fi
}

case "$ACTION" in
    start)   start_tunnel ;;
    stop)    stop_tunnel ;;
    status)  status_tunnel ;;
    restart) stop_tunnel; sleep 1; start_tunnel ;;
    *)
        echo "Uso: $0 {start|stop|status|restart} [postgres] <tunnel-host> [local-port]"
        exit 1
        ;;
esac
