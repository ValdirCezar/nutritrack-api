# 🥗 NutriTrack AI — Documentação do Projeto

## Visão Geral

NutriTrack AI é um app de acompanhamento nutricional que usa inteligência artificial (OpenAI) para analisar refeições descritas em linguagem natural e calcular automaticamente calorias e macronutrientes.

**Stack:**
- **Backend:** Go 1.24 (net/http nativo, sem framework)
- **Frontend:** Angular 19 (standalone components, signals)
- **Banco de Dados:** MongoDB 8.0 (Atlas em produção, Docker local)
- **IA:** OpenAI API (modelo gpt-4o-mini)
- **E-mail:** SMTP via Gmail (verificação e reset de senha)

---

## 📁 Estrutura dos Repositórios

### Monorepo local
```
~/Desktop/nutritrack-ai/
├── docker-compose.yml          # MongoDB local para desenvolvimento
├── nutritrack-api/             # Backend (Go)
└── nutritrack-web/             # Frontend (Angular)
```

### GitHub
| Repositório | URL |
|---|---|
| Backend | https://github.com/ValdirCezar/nutritrack-api |
| Frontend | https://github.com/ValdirCezar/nutritrack-web |

---

## 🏗️ Arquitetura do Backend

```
nutritrack-api/
├── cmd/server/main.go              # Entrypoint — rotas, DI, servidor HTTP
├── internal/
│   ├── config/config.go            # Configuração via env vars + conexão MongoDB
│   ├── handler/                    # HTTP handlers (controllers)
│   │   ├── auth_handler.go         # Registro, login, verificação, reset
│   │   ├── dashboard_handler.go    # Dashboard com totais do dia
│   │   ├── meal_handler.go         # CRUD de refeições
│   │   ├── profile_handler.go      # Perfil do usuário (metas)
│   │   └── response.go            # Helpers de resposta JSON
│   ├── middleware/
│   │   ├── auth.go                 # JWT middleware
│   │   └── cors.go                 # CORS configuration
│   ├── model/                      # Structs do MongoDB
│   │   ├── user.go                 # User + VerificationCode
│   │   ├── meal.go                 # Meal + Food
│   │   ├── profile.go              # Profile (metas nutricionais)
│   │   └── food_cache.go           # Cache de alimentos já analisados
│   ├── repository/                 # Acesso ao MongoDB (queries)
│   └── service/                    # Lógica de negócio
│       ├── auth_service.go         # Auth + JWT + verificação
│       ├── email_service.go        # Envio de e-mails SMTP
│       ├── meal_service.go         # Registro + análise de refeições
│       ├── openai_service.go       # Integração com OpenAI
│       ├── food_cache_service.go   # Cache inteligente de alimentos
│       └── profile_service.go      # Perfil/metas do usuário
├── Dockerfile                      # Build multi-stage (Go → Alpine)
├── render.yaml                     # Blueprint para deploy no Render
├── go.mod / go.sum
└── .env                            # Variáveis locais (NÃO commitado)
```

### API Endpoints

| Método | Rota | Auth | Descrição |
|--------|------|------|-----------|
| `GET` | `/api/health` | ❌ | Health check (Render) |
| `POST` | `/api/auth/register` | ❌ | Criar conta (envia código por e-mail) |
| `POST` | `/api/auth/verify-email` | ❌ | Verificar e-mail com código de 6 dígitos |
| `POST` | `/api/auth/resend-code` | ❌ | Reenviar código (cooldown 60s) |
| `POST` | `/api/auth/login` | ❌ | Login (retorna JWT) |
| `POST` | `/api/auth/forgot-password` | ❌ | Solicitar reset de senha |
| `POST` | `/api/auth/reset-password` | ❌ | Redefinir senha com código |
| `GET` | `/api/auth/me` | ✅ | Dados do usuário logado |
| `PUT` | `/api/auth/password` | ✅ | Alterar senha |
| `PUT` | `/api/auth/email` | ✅ | Alterar e-mail |
| `POST` | `/api/profile` | ✅ | Criar/atualizar perfil (metas) |
| `GET` | `/api/profile` | ✅ | Obter perfil |
| `POST` | `/api/meals` | ✅ | Registrar refeição (análise via IA) |
| `GET` | `/api/meals?date=YYYY-MM-DD` | ✅ | Listar refeições do dia |
| `DELETE` | `/api/meals/{id}` | ✅ | Deletar refeição |
| `GET` | `/api/dashboard?date=YYYY-MM-DD` | ✅ | Dashboard (refeições + totais + metas) |

---

## 🎨 Arquitetura do Frontend

```
nutritrack-web/
├── src/
│   ├── app/
│   │   ├── app.component.ts           # Root component
│   │   ├── app.config.ts              # Providers (HttpClient, Router)
│   │   ├── app.routes.ts              # Rotas da aplicação
│   │   ├── core/
│   │   │   ├── guards/auth.guard.ts   # Proteção de rotas autenticadas
│   │   │   ├── interceptors/auth.interceptor.ts  # Injeta JWT nos requests
│   │   │   ├── models/                # Interfaces TypeScript
│   │   │   └── services/
│   │   │       ├── api.service.ts     # HttpClient para o backend
│   │   │       └── auth.service.ts    # Login, registro, token management
│   │   └── features/
│   │       ├── auth/
│   │       │   ├── login/             # Tela de login
│   │       │   ├── register/          # Tela de registro
│   │       │   ├── forgot-password/   # Reset de senha (3 etapas)
│   │       │   └── verify-email/      # Verificação por código
│   │       ├── dashboard/             # Dashboard principal + voice input
│   │       ├── history/               # Histórico por data
│   │       └── onboarding/            # Setup inicial de metas
│   ├── environments/
│   │   ├── environment.ts             # Dev: http://localhost:8080/api
│   │   └── environment.prod.ts        # Prod: https://nutritrack-api-9yc3.onrender.com/api
│   ├── styles.scss                    # CSS global + variáveis de tema
│   └── index.html
├── netlify.toml                       # Configuração de build Netlify
├── public/_redirects                  # SPA redirect para Netlify
├── angular.json
└── package.json
```

### Rotas do Frontend

| Rota | Componente | Auth | Descrição |
|------|-----------|------|-----------|
| `/login` | LoginComponent | ❌ | Tela de login |
| `/register` | RegisterComponent | ❌ | Tela de registro |
| `/forgot-password` | ForgotPasswordComponent | ❌ | Reset de senha (3 etapas) |
| `/verify-email` | VerifyEmailComponent | ❌ | Verificação de e-mail |
| `/onboarding` | OnboardingComponent | ✅ | Setup de metas |
| `/dashboard` | DashboardComponent | ✅ | Dashboard + registro de refeições |
| `/history` | HistoryComponent | ✅ | Histórico por data |
| `/` | → redirect `/dashboard` | — | — |

---

## ☁️ Infraestrutura de Produção

### Mapa de Serviços

```
┌──────────────────────────────────────┐
│           NETLIFY (Frontend)         │
│  https://nutritrack-ai-web.netlify.app│
│  Angular 19 — Build estático (SPA)   │
└──────────────┬───────────────────────┘
               │ HTTPS requests
               ▼
┌──────────────────────────────────────┐
│           RENDER (Backend)           │
│  https://nutritrack-api-9yc3.onrender.com │
│  Go 1.24 — Docker (Alpine)          │
│  Plano: Free (dorme após 15min)     │
│  Região: Oregon                      │
└──────────┬───────────┬───────────────┘
           │           │
           ▼           ▼
┌────────────────┐  ┌──────────────┐
│ MongoDB Atlas  │  │  OpenAI API  │
│ M0 Free (SP)   │  │  gpt-4o-mini │
│ nutritrack-    │  └──────────────┘
│ cluster        │
└────────────────┘
```

### IDs e URLs Importantes

| Serviço | Detalhe |
|---------|---------|
| **Render Service ID** | `srv-d6nh5dea2pns738qgsk0` |
| **Render Dashboard** | https://dashboard.render.com/web/srv-d6nh5dea2pns738qgsk0 |
| **Render URL** | https://nutritrack-api-9yc3.onrender.com |
| **Netlify Project ID** | `09876f3f-4df6-4a94-bf6a-d199b74e6f65` |
| **Netlify Admin** | https://app.netlify.com/projects/nutritrack-ai-web |
| **Netlify URL** | https://nutritrack-ai-web.netlify.app |
| **Atlas Project ID** | `69af11835e8d145f95bb5676` |
| **Atlas Project** | NutriTrack |
| **Atlas Cluster** | nutritrack-cluster (M0 Free, AWS SA_EAST_1) |
| **Atlas Connection** | `mongodb+srv://nutritrack-cluster.klwywzh.mongodb.net` |
| **GitHub Backend** | https://github.com/ValdirCezar/nutritrack-api |
| **GitHub Frontend** | https://github.com/ValdirCezar/nutritrack-web |

---

## 🔧 Variáveis de Ambiente (Backend)

Configuradas no Render. Para alterar, use o CLI ou dashboard.

| Variável | Descrição | Sensível |
|----------|-----------|----------|
| `MONGO_URI` | Connection string do MongoDB Atlas | ✅ |
| `MONGO_DATABASE` | Nome do banco (`nutritrack`) | ❌ |
| `JWT_SECRET` | Segredo para assinar tokens JWT (32+ chars) | ✅ |
| `OPENAI_API_KEY` | Chave da API OpenAI | ✅ |
| `OPENAI_MODEL` | Modelo da OpenAI (`gpt-4o-mini`) | ❌ |
| `CORS_ORIGIN` | URL do frontend permitida pelo CORS | ❌ |
| `SMTP_HOST` | Host SMTP (`smtp.gmail.com`) | ❌ |
| `SMTP_PORT` | Porta SMTP (`587`) | ❌ |
| `SMTP_USER` | E-mail do remetente | ✅ |
| `SMTP_PASS` | App password do Gmail | ✅ |
| `SMTP_FROM` | Nome + e-mail do remetente | ❌ |
| `PORT` | Porta do servidor (injetada pelo Render) | ❌ |

---

## 🗄️ Banco de Dados — MongoDB

### Collections

| Collection | Descrição |
|-----------|-----------|
| `users` | Usuários (email, senha hash, verificado) |
| `verification_codes` | Códigos de verificação (6 dígitos, expira 10min) |
| `password_reset_tokens` | Tokens de reset de senha |
| `profiles` | Perfis (metas: calorias, proteína, carbs, gordura) |
| `meals` | Refeições (descrição, foods[], totals, data) |
| `food_cache` | Cache de alimentos já analisados pela IA |

---

## 🛠️ Guia de Operações via CLI

### Pré-requisitos (CLIs instalados)

```bash
# Verificar se todos os CLIs estão disponíveis
gh auth status          # GitHub CLI
atlas auth whoami       # MongoDB Atlas CLI
netlify status          # Netlify CLI (executar dentro de nutritrack-web/)
```

Se algum não estiver autenticado:
```bash
gh auth login           # GitHub — abre navegador
atlas auth login        # Atlas — abre navegador com código
netlify login           # Netlify — abre navegador
```

---

### 🚀 Deploy do Backend (Render)

O deploy é **automático** a cada push na branch `main` do GitHub.

```bash
# Fazer uma alteração e deploy
cd ~/Desktop/nutritrack-ai/nutritrack-api
# ... editar código ...
git add -A
git commit -m "feat: sua mensagem"
git push origin main
# O Render detecta o push e faz build + deploy automaticamente
```

**Verificar status do deploy:**
```bash
# Via API do Render
curl -s -H "Authorization: Bearer SUA_RENDER_API_KEY" \
  "https://api.render.com/v1/services/srv-d6nh5dea2pns738qgsk0/deploys?limit=1" \
  | python3 -m json.tool
```

**Testar se o backend está no ar:**
```bash
curl https://nutritrack-api-9yc3.onrender.com/api/health
# Resposta esperada: {"status":"ok"}
# Nota: se estiver dormindo (free tier), a primeira request demora ~30s
```

**Ver/alterar variáveis de ambiente do Render:**
```bash
# LISTAR todas as env vars
curl -s -H "Authorization: Bearer SUA_RENDER_API_KEY" \
  "https://api.render.com/v1/services/srv-d6nh5dea2pns738qgsk0/env-vars" \
  | python3 -m json.tool

# ALTERAR uma env var (ex: CORS_ORIGIN)
curl -s -X PUT \
  "https://api.render.com/v1/services/srv-d6nh5dea2pns738qgsk0/env-vars/CORS_ORIGIN" \
  -H "Authorization: Bearer SUA_RENDER_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"value": "https://novo-dominio.netlify.app"}'
```

**Forçar redeploy manual:**
```bash
curl -s -X POST \
  "https://api.render.com/v1/services/srv-d6nh5dea2pns738qgsk0/deploys" \
  -H "Authorization: Bearer SUA_RENDER_API_KEY" \
  -H "Content-Type: application/json"
```

**Dashboard web:** https://dashboard.render.com/web/srv-d6nh5dea2pns738qgsk0

---

### 🌐 Deploy do Frontend (Netlify)

O deploy pode ser feito via CLI ou automaticamente via GitHub (se configurado).

**Deploy manual via CLI:**
```bash
cd ~/Desktop/nutritrack-ai/nutritrack-web

# Build de produção
npx ng build --configuration production

# Deploy para produção
netlify deploy --prod --dir=dist/nutritrack-web/browser

# Deploy de preview (para testar antes de ir para prod)
netlify deploy --dir=dist/nutritrack-web/browser
```

**Conectar auto-deploy via GitHub (recomendado):**
```bash
# Dentro da pasta do frontend
netlify link    # se não estiver linkado
netlify init    # configura CI/CD com GitHub
```

Ou configure direto no painel: https://app.netlify.com/projects/nutritrack-ai-web → Site Configuration → Build & deploy → Link to Git

**Configurações de build no Netlify:**
- **Build command:** `npm run build`
- **Publish directory:** `dist/nutritrack-web/browser`
- **Node version:** 22 (definir em Environment variables: `NODE_VERSION=22`)

**Dashboard web:** https://app.netlify.com/projects/nutritrack-ai-web

---

### 🍃 MongoDB Atlas (Banco de Dados)

**Conectar ao banco de produção via mongosh:**
```bash
mongosh "mongodb+srv://nutritrack-cluster.klwywzh.mongodb.net/nutritrack" \
  --username nutritrack-admin \
  --password 'NtrkProd2026Secure'
```

**Consultas úteis:**
```javascript
// Listar todos os usuários
db.users.find({}, { email: 1, name: 1, emailVerified: 1 }).pretty()

// Ver refeições de um usuário em uma data
db.meals.find({ userId: ObjectId("..."), date: "2026-03-09" }).pretty()

// Ver perfil de um usuário
db.profiles.find({ userId: ObjectId("...") }).pretty()

// Contar documentos por collection
db.getCollectionNames().forEach(c => print(c + ": " + db[c].countDocuments()))

// Verificar manualmente o e-mail de um usuário
db.users.updateOne({ email: "user@example.com" }, { $set: { emailVerified: true } })

// Deletar um usuário e todos os dados associados
const user = db.users.findOne({ email: "user@example.com" })
if (user) {
  db.meals.deleteMany({ userId: user._id })
  db.profiles.deleteOne({ userId: user._id })
  db.verification_codes.deleteMany({ email: user.email })
  db.password_reset_tokens.deleteMany({ email: user.email })
  db.users.deleteOne({ _id: user._id })
  print("Usuário e dados removidos")
}

// Limpar TODA a base (cuidado!)
db.dropDatabase()
```

**Gerenciar via Atlas CLI:**
```bash
# Listar clusters
atlas clusters list --projectId 69af11835e8d145f95bb5676

# Ver status do cluster
atlas clusters describe nutritrack-cluster --projectId 69af11835e8d145f95bb5676

# Listar database users
atlas dbusers list --projectId 69af11835e8d145f95bb5676

# Criar novo database user
atlas dbusers create --projectId 69af11835e8d145f95bb5676 \
  --username novo-user --password 'SenhaSegura123' --role readWriteAnyDatabase

# Ver IP access list
atlas accessLists list --projectId 69af11835e8d145f95bb5676
```

**Dashboard web:** https://cloud.mongodb.com (Projeto: NutriTrack)

---

## 💻 Desenvolvimento Local

### Setup inicial

```bash
# 1. Clonar os repos
cd ~/Desktop/nutritrack-ai
git clone https://github.com/ValdirCezar/nutritrack-api.git
git clone https://github.com/ValdirCezar/nutritrack-web.git

# 2. Subir MongoDB local
docker compose up -d

# 3. Configurar backend
cd nutritrack-api
cp .env.example .env   # ou criar manualmente (ver seção de env vars)
# Editar .env com suas credenciais locais

# 4. Rodar backend
go run ./cmd/server/
# Saída esperada: "NutriTrack API rodando na porta 8080"

# 5. Instalar e rodar frontend (em outro terminal)
cd ../nutritrack-web
npm install
npx ng serve --port 4200
# Acesse: http://localhost:4200
```

### .env local do backend (exemplo)

```env
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=nutritrack
JWT_SECRET=dev-secret-pelo-menos-32-caracteres-aqui
OPENAI_API_KEY=sk-proj-sua-chave-aqui
OPENAI_MODEL=gpt-4o-mini
CORS_ORIGIN=http://localhost:4200
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=seu-email@gmail.com
SMTP_PASS=sua-app-password
SMTP_FROM=NutriTrack AI <seu-email@gmail.com>
```

### Limpar banco local

```bash
docker exec nutritrack-mongo mongosh nutritrack --eval "db.dropDatabase()"
```

### Reiniciar MongoDB local

```bash
docker compose down && docker compose up -d
```

---

## 🔄 Fluxos Importantes

### Registro de Usuário
1. POST `/api/auth/register` → cria user com `emailVerified: false` → envia código por e-mail
2. POST `/api/auth/verify-email` → valida código → seta `emailVerified: true` → retorna JWT
3. Frontend redireciona para `/onboarding` (setup de metas)

### Reset de Senha
1. POST `/api/auth/forgot-password` → envia código por e-mail
2. POST `/api/auth/reset-password` → valida código + nova senha → atualiza senha

### Registro de Refeição
1. Usuário digita (ou fala) descrição: "2 ovos, 100g de arroz e salada"
2. POST `/api/meals` → backend envia para OpenAI → recebe análise nutricional
3. Resultado salvo no MongoDB com foods[] detalhados e totais
4. Cache inteligente: alimentos já analisados são reutilizados sem chamar a IA

---

## ⚠️ Observações Importantes

- **Render Free Tier:** O backend dorme após 15 minutos de inatividade. A primeira request após dormir leva ~30 segundos (cold start). Para uso em produção real, considerar upgrade para plano pago.
- **MongoDB Atlas M0:** Limite de 512MB de storage. Suficiente para MVP, mas monitorar uso.
- **OpenAI API:** Tem custo por uso. O modelo `gpt-4o-mini` é o mais barato. O cache de alimentos (`food_cache`) reduz chamadas repetidas.
- **SMTP Gmail:** Usar App Password (não a senha normal). Configurar em: Google Account → Security → 2-Step Verification → App passwords.
- **CORS:** O backend só aceita requests da origem configurada em `CORS_ORIGIN`. Ao mudar o domínio do frontend, atualizar essa variável no Render.

---

## 📋 Checklist de Troubleshooting

| Problema | Solução |
|----------|---------|
| Backend não responde | Pode estar dormindo (free tier). Espere ~30s ou faça curl no health check |
| CORS error no browser | Verificar se `CORS_ORIGIN` no Render bate com a URL do Netlify |
| E-mail não chega | Verificar `SMTP_USER`, `SMTP_PASS` e se a App Password do Gmail está ativa |
| Login retorna "e-mail não verificado" | Verificar e-mail via `db.users.updateOne(...)` ou reenviar código |
| OpenAI retorna erro | Verificar se `OPENAI_API_KEY` está válida e com créditos |
| Build do Angular falha | Rodar `npm install` e verificar versão do Node (22+) |
| MongoDB connection timeout | Verificar se IP `0.0.0.0/0` está na access list do Atlas |
