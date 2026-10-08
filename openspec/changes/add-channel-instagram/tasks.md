# Tasks

> Depende de `add-channel-abstraction` fases 0 a 4. **A entrega para clientes reais depende de App Review + Business Verification.**

## 0. Bloqueios (iniciar já, em paralelo)
- [ ] 0.1 **Reler a documentação atual** da Meta (Instagram Platform: messaging, webhooks, permissões, limites, versão da
      Graph API): a leitura que baseia esta change não foi reconfirmada
- [ ] 0.2 **Criar o app Meta e iniciar App Review e Business Verification** (Advanced Access para
      `instagram_business_manage_messages`); registrar o prazo real
- [ ] 0.3 Preparar o que o review exige: política de privacidade, vídeo de demonstração do fluxo, caso de uso por permissão
- [ ] 0.4 Decidir a rota (Instagram Login x Messenger) e se a rota por Página também será implementada

## 1. Backend
- [ ] 1.1 OAuth Business Login (code → curta → longa duração) e job de refresh; `status=reauth`
- [ ] 1.2 Webhook: `GET` de verificação e `POST` com `X-Hub-Signature-256` (App Secret) em tempo constante; dedup por `mid`
- [ ] 1.3 Adaptador `instagram` (`ParseInbound`, `Send` texto/mídia/quick replies/templates, resposta privada a comentário)
- [ ] 1.4 `ReplyWindow` 24 h + `human_agent` 7 d: `CanReply`, `replyPolicy`, recusa `409`
- [ ] 1.5 Mídia: baixar na chegada (URL expira); respeitar tipos e tamanhos (imagem 8 MB, áudio/vídeo/PDF 25 MB)
- [ ] 1.6 Versão da Graph API configurável; token-bucket por conta (100/s texto, 10/s áudio/vídeo, 750/h respostas privadas)
- [ ] 1.7 Aviso de divulgação de bot quando houver automação respondendo

## 2. Frontend
- [ ] 2.1 Botão "Conectar com Instagram" + estado "aguardando aprovação do app" quando o Advanced Access não existir
- [ ] 2.2 Composer com contagem regressiva de 24 h, ação "enviar como atendente humano" e contador de 1000 bytes
- [ ] 2.3 Origem da conversa (anúncio/post/comentário) no cabeçalho; badge e token `channel-instagram`; traduções

## 3. Verificação
- [ ] 3.1 Contrato + servidor Meta simulado
- [ ] 3.2 Mutação nos testes de segurança (assinatura, desafio, token cifrado, dedup)
- [ ] 3.3 Validação ao vivo com contas com papel no app (antes do review) e depois com contas de terceiros
- [ ] 3.4 Documentação de usuário e `/qa-analyst`
