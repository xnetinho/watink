# Runbook Operacional: Backup e Restauração de Desastre (Watink)

## 1. Visão Geral
Este runbook documenta os procedimentos de backup e recuperação contra desastres (Disaster Recovery) para o ambiente de banco de dados do Watink.
- **RPO Alvo**: < 1 hora (backups lógicos periódicos com retenção).
- **RTO Alvo**: < 15 minutos (restauração limpa e revalidação de RLS).

## 2. Procedimento de Backup

### 2.1 Execução Manual
```bash
DB_HOST=localhost DB_PORT=5432 DB_NAME=watink DB_PASS=sua_senha ./scripts/backup-db.sh
```
O script gera um arquivo formato custom (`-Fc`) compactado em `/tmp/watink-backups/watink_backup_YYYYMMDD_HHMMSS.dump`.

### 2.2 Automação Recomendada (Cron)
Agendamento diário com rotação de 7 dias:
```cron
0 2 * * * /root/watink/scripts/backup-db.sh >> /var/log/watink-backup.log 2>&1
```

## 3. Procedimento de Restauração

### 3.1 Restauração em Banco Limpo
1. Certifique-se de que o container Postgres esteja em execução.
2. Execute o script de restauração especificando o arquivo e banco:
   ```bash
   ./scripts/restore-db.sh /caminho/do/backup.dump watink
   ```
3. Execute o job de migração para revalidar/restabelecer políticas de RLS e papéis restritos:
   ```bash
   docker compose run --rm watink-migrate
   ```

## 4. Teste de Validação
O procedimento é auditado automaticamente através da criação de banco efêmero e round-trip completo de dump e restauração.
EOF
