#!/bin/bash

# Script de gestão de SSH tunnel para bases de dados remotas (PostgreSQL e MySQL)
# Uso: ./scripts/db-tunnel.sh {start|stop|status|restart} [postgres|mysql]

DB_TYPE="${2:-postgres}"
TUNNEL_HOST="rswebportal.pt"
REMOTE_HOST="localhost"

# Função auxiliar para testar conectividade da porta
test_port_connectivity() {
    local port=$1
    local timeout=${2:-2}
    
    # Tentar usar nc (netcat) se disponível
    if command -v nc >/dev/null 2>&1; then
        timeout "$timeout" nc -z localhost "$port" >/dev/null 2>&1
        return $?
    elif command -v timeout >/dev/null 2>&1; then
        timeout "$timeout" bash -c "</dev/tcp/localhost/$port" >/dev/null 2>&1
        return $?
    else
        # Fallback simples
        (echo >/dev/tcp/localhost/"$port") >/dev/null 2>&1
        return $?
    fi
}

# Função para verificar se túnel está realmente funcional
check_tunnel_health() {
    local pid=$1
    
    # Verificar se processo existe
    if ! ps -p "$pid" > /dev/null 2>&1; then
        return 1
    fi
    
    # Verificar se porta está a ouvir
    if ! lsof -i :"$LOCAL_PORT" -sTCP:LISTEN >/dev/null 2>&1; then
        return 1
    fi
    
    # Verificar conectividade da porta
    if ! test_port_connectivity "$LOCAL_PORT" 2; then
        return 1
    fi
    
    return 0
}

# Configuração por tipo de base de dados
case "$DB_TYPE" in
    postgres)
        REMOTE_PORT="5432"
        LOCAL_PORT="5432"
        PID_FILE="/tmp/postgres-tunnel-5432.pid"
        TEST_CMD="pg_isready"
        TEST_ARGS="-h localhost -p $LOCAL_PORT -U postgres"
        ;;
    mysql)
        REMOTE_PORT="3306"
        LOCAL_PORT="3306"
        PID_FILE="/tmp/mysql-tunnel-3306.pid"
        TEST_CMD="mysqladmin"
        TEST_ARGS="ping -h localhost -P $LOCAL_PORT"
        ;;
    *)
        echo "Error: Invalid database type. Use 'postgres' or 'mysql'"
        exit 1
        ;;
esac

start_tunnel() {
    # Verificar se ficheiro PID existe e processo está ativo
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if [ -n "$PID" ] && ps -p "$PID" > /dev/null 2>&1; then
            # Processo existe, mas verificar se túnel está funcional
            if check_tunnel_health "$PID"; then
                echo "$DB_TYPE tunnel already running (PID: $PID)"
                return 0
            else
                echo "Found stale tunnel process (PID: $PID), cleaning up..."
                kill "$PID" 2>/dev/null
                rm "$PID_FILE"
                sleep 1
            fi
        else
            rm "$PID_FILE"
        fi
    fi
    
    # Verificar se a porta já está em uso por outro processo
    if lsof -i :"$LOCAL_PORT" >/dev/null 2>&1 || ss -tln | grep -q ":$LOCAL_PORT "; then
        # Tentar encontrar o PID do SSH que está a usar a porta
        EXISTING_PID=$(lsof -ti :"$LOCAL_PORT" 2>/dev/null | head -n1)
        if [ -n "$EXISTING_PID" ]; then
            # Verificar se é um processo SSH
            if ps -p "$EXISTING_PID" -o comm= 2>/dev/null | grep -q ssh; then
                # Verificar se túnel está funcional
                if check_tunnel_health "$EXISTING_PID"; then
                    echo "Found existing SSH tunnel on port $LOCAL_PORT (PID: $EXISTING_PID)"
                    echo "$EXISTING_PID" > "$PID_FILE"
                    echo "$DB_TYPE tunnel already active (PID: $EXISTING_PID)"
                    return 0
                else
                    echo "Found non-functional SSH tunnel (PID: $EXISTING_PID), restarting..."
                    kill "$EXISTING_PID" 2>/dev/null
                    sleep 2
                fi
            fi
        fi
        
        # Se ainda houver algo na porta, reportar erro
        if lsof -i :"$LOCAL_PORT" >/dev/null 2>&1; then
            echo "Error: Port $LOCAL_PORT is already in use by another process"
            return 1
        fi
    fi
    
    echo "Starting SSH tunnel for $DB_TYPE to $TUNNEL_HOST..."
    ssh -f -N -L "$LOCAL_PORT:$REMOTE_HOST:$REMOTE_PORT" "$TUNNEL_HOST"
    
    if [ $? -ne 0 ]; then
        echo "Failed to start SSH tunnel"
        return 1
    fi
    
    # Aguardar um pouco para o túnel se estabelecer
    sleep 2
    
    # Capturar o PID do processo SSH que acabou de ser criado
    SSH_PID=$(lsof -ti :"$LOCAL_PORT" 2>/dev/null | head -n1)
    
    if [ -z "$SSH_PID" ]; then
        echo "Warning: Could not determine SSH tunnel PID"
        return 1
    fi
    
    echo "$SSH_PID" > "$PID_FILE"
    echo "$DB_TYPE tunnel started with PID: $SSH_PID"
    
    # Testar conectividade do túnel
    if test_port_connectivity "$LOCAL_PORT" 3; then
        echo "✓ Tunnel port $LOCAL_PORT is responding"
        
        # Test database connection se ferramenta específica disponível
        if command -v "$TEST_CMD" >/dev/null 2>&1; then
            if $TEST_CMD $TEST_ARGS >/dev/null 2>&1; then
                echo "✓ $DB_TYPE database is accessible via tunnel"
            else
                echo "⚠ Tunnel active but $DB_TYPE database not responding"
            fi
        fi
    else
        echo "⚠ Warning: Tunnel started but port not responding yet"
    fi
    
    return 0
}

stop_tunnel() {
    if [ ! -f "$PID_FILE" ]; then
        echo "No $DB_TYPE tunnel PID file found (tunnel not running)"
        return 0
    fi
    
    PID=$(cat "$PID_FILE")
    echo "Stopping $DB_TYPE tunnel (PID: $PID)..."
    
    if kill "$PID" 2>/dev/null; then
        echo "$DB_TYPE tunnel stopped"
        rm "$PID_FILE"
    else
        echo "Failed to stop $DB_TYPE tunnel (PID: $PID) - removing stale PID file"
        rm "$PID_FILE"
    fi
    
    return 0
}

status_tunnel() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            echo "$DB_TYPE tunnel process is running (PID: $PID)"
            
            # Verificar saúde do túnel
            if check_tunnel_health "$PID"; then
                echo "✓ Tunnel is healthy and port $LOCAL_PORT is responding"
                
                # Test database connection
                if command -v "$TEST_CMD" >/dev/null 2>&1; then
                    if $TEST_CMD $TEST_ARGS >/dev/null 2>&1; then
                        echo "✓ $DB_TYPE database is accessible"
                    else
                        echo "⚠ Tunnel active but database not responding"
                    fi
                else
                    # Teste alternativo de conectividade
                    if test_port_connectivity "$LOCAL_PORT" 2; then
                        echo "✓ Port $LOCAL_PORT is accepting connections"
                    fi
                fi
            else
                echo "✗ Tunnel process exists but is NOT functional (may need restart)"
                return 1
            fi
        else
            echo "$DB_TYPE tunnel PID file exists but process is not running"
            rm "$PID_FILE"
            return 1
        fi
    else
        echo "$DB_TYPE tunnel is not running"
        return 1
    fi
}

case "$1" in
    start)
        start_tunnel
        ;;
    stop)
        stop_tunnel
        ;;
    status)
        status_tunnel
        ;;
    restart)
        stop_tunnel
        sleep 1
        start_tunnel
        ;;
    *)
        echo "Usage: $0 {start|stop|status|restart} [postgres|mysql]"
        echo "  Default database type: postgres"
        exit 1
        ;;
esac
