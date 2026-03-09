# NutriTrack API

Backend da aplicação NutriTrack AI — acompanhamento nutricional com inteligência artificial.

**Stack:** Go 1.24 · MongoDB · OpenAI API · SMTP

📖 **Documentação completa:** Veja o [PROJETO.md](https://github.com/ValdirCezar/nutritrack-api/blob/main/PROJETO.md) na raiz para guia de deploy, operações e arquitetura.

## Executar localmente

```bash
# Subir MongoDB
docker compose -f ../docker-compose.yml up -d

# Configurar variáveis
cp .env.example .env  # editar com suas credenciais

# Rodar
go run ./cmd/server/
```

## Deploy

O deploy é automático via push na branch `main` para o **Render**.

```bash
git push origin main  # Render faz build + deploy automaticamente
```

## API

`GET /api/health` · `POST /api/auth/register` · `POST /api/auth/login` · `POST /api/meals` · `GET /api/dashboard` · [ver todos os endpoints no PROJETO.md]
