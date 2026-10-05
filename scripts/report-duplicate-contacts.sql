-- Relatório (SOMENTE LEITURA) de contatos que parecem ser a MESMA pessoa em dois
-- cadastros: um pelo telefone (agenda) e outro criado a partir de um LID
-- ("...@lid") antes da unificação por telefone existir.
--
-- Não altera nada. Mesclar mexe em tickets, mensagens e vínculos de Cliente, por
-- isso a decisão fica com quem revisar a lista.
--
-- Uso:  psql "$DATABASE_URL" -f scripts/report-duplicate-contacts.sql
--
-- Como casa: o contato "órfão" tem number = parte numérica do próprio LID (era o
-- que o sistema gravava quando só conhecia o LID). O mapa LID -> telefone vem da
-- tabela whatsmeow_lid_map (mantida pelo engine), restrita ao mesmo tenant.

WITH lid_contacts AS (
    SELECT c.id, c."tenantId", c.name, c.number,
           regexp_replace(COALESCE(c.lid, c.number), '@lid$', '') AS lid_num
    FROM "Contacts" c
    WHERE c."isGroup" = false
      AND (c.lid IS NOT NULL OR c.number !~ '^[0-9]{10,13}$')
),
pairs AS (
    SELECT lc."tenantId",
           lc.id          AS lid_contact_id,
           lc.name        AS lid_contact_name,
           lc.number      AS lid_contact_number,
           pc.id          AS phone_contact_id,
           pc.name        AS phone_contact_name,
           pc.number      AS phone_number
    FROM lid_contacts lc
    JOIN whatsmeow_lid_map m ON m.lid = lc.lid_num
    JOIN "Contacts" pc
      ON pc."tenantId" = lc."tenantId"
     AND pc."isGroup" = false
     AND pc.number = m.pn
     AND pc.id <> lc.id
)
SELECT p."tenantId",
       p.phone_contact_id, p.phone_contact_name, p.phone_number,
       p.lid_contact_id,   p.lid_contact_name,   p.lid_contact_number,
       (SELECT count(*) FROM "Tickets"  t WHERE t."contactId" = p.phone_contact_id) AS tickets_phone,
       (SELECT count(*) FROM "Tickets"  t WHERE t."contactId" = p.lid_contact_id)   AS tickets_lid,
       (SELECT count(*) FROM "Messages" x WHERE x."contactId" = p.phone_contact_id) AS msgs_phone,
       (SELECT count(*) FROM "Messages" x WHERE x."contactId" = p.lid_contact_id)   AS msgs_lid
FROM pairs p
ORDER BY p."tenantId", p.phone_contact_id;
