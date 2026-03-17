package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/valdircezar/nutritrack-api/internal/model"
)

// OpenAIService gerencia a comunicação com a API da OpenAI para análise nutricional
type OpenAIService struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// openAIChatRequest representa o corpo da requisição para a API de chat da OpenAI
type openAIChatRequest struct {
	Model          string            `json:"model"`
	Messages       []openAIChatMsg   `json:"messages"`
	ResponseFormat openAIRespFormat  `json:"response_format"`
	Temperature    float64           `json:"temperature"`
	Seed           int               `json:"seed"`
}

// openAIChatMsg representa uma mensagem no formato do chat da OpenAI
type openAIChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIRespFormat especifica o formato de resposta desejado
type openAIRespFormat struct {
	Type string `json:"type"`
}

// openAIChatResponse representa a resposta da API de chat da OpenAI
type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// NewOpenAIService cria uma nova instância do serviço de integração com a OpenAI
func NewOpenAIService(apiKey, model string) *OpenAIService {
	return &OpenAIService{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AnalyzeFood envia a descrição do alimento para a OpenAI e retorna os dados nutricionais.
// Utiliza response_format JSON para garantir uma resposta estruturada.
func (s *OpenAIService) AnalyzeFood(description string) (*model.OpenAIFoodResponse, error) {
	if description == "" {
		return nil, errors.New("descrição do alimento não pode ser vazia")
	}

	// Prompt do sistema em português instruindo a OpenAI a agir como nutricionista
	systemPrompt := `Você é um nutricionista especializado em composição de alimentos. Analise os alimentos descritos e retorne APENAS JSON no formato especificado. Sem texto extra.

IMPORTANTE — Base de dados nutricional:
- Use como referência principal a Tabela TACO (Tabela Brasileira de Composição de Alimentos) da UNICAMP/NEPA.
- Como referência secundária, use o USDA FoodData Central (para alimentos não encontrados na TACO).
- NÃO superestime nem subestime valores. Use as médias da tabela de referência.

CÁLCULO PROPORCIONAL OBRIGATÓRIO:
Você DEVE seguir estes passos para CADA alimento:
1. Identifique o valor nutricional de referência por 100g (da TACO ou USDA).
2. Calcule o fator de escala: fator = quantidade_informada_em_gramas / 100.
3. Multiplique CADA nutriente pelo fator: valor_final = valor_por_100g × fator.
4. Confira: se a quantidade for maior que 100g, TODOS os valores DEVEM ser maiores que os de 100g.

Referências TACO por 100g (use como base de cálculo):
- Ovo cozido (100g): 13.0g proteína, 0.6g carboidrato, 8.5g gordura, 146 kcal (1 unidade ≈ 50g)
- Arroz branco cozido (100g): 2.5g proteína, 28.1g carboidrato, 0.2g gordura, 128 kcal
- Peito de frango grelhado (100g): 31.5g proteína, 0g carboidrato, 1.3g gordura, 159 kcal
- Feijão carioca cozido (100g): 4.8g proteína, 13.6g carboidrato, 0.5g gordura, 76 kcal
- Banana prata (100g): 1.3g proteína, 26.0g carboidrato, 0.1g gordura, 98 kcal (1 unidade ≈ 86g)
- Aveia em flocos (100g): 13.9g proteína, 66.6g carboidrato, 8.5g gordura, 394 kcal
- Batata doce cozida (100g): 1.3g proteína, 18.4g carboidrato, 0.1g gordura, 77 kcal
- Macarrão cozido (100g): 3.4g proteína, 19.9g carboidrato, 0.5g gordura, 102 kcal

EXEMPLO DE CÁLCULO (siga este modelo):
Entrada: "180g de aveia em flocos"
→ Referência TACO para aveia em flocos (100g): 13.9g prot, 66.6g carb, 8.5g gord, 394 kcal
→ Fator de escala: 180 / 100 = 1.8
→ Proteína: 13.9 × 1.8 = 25.0g
→ Carboidrato: 66.6 × 1.8 = 119.9g
→ Gordura: 8.5 × 1.8 = 15.3g
→ Calorias: 394 × 1.8 = 709.2 kcal

O formato JSON deve ser:
{
  "foods": [
    {"name": "nome do alimento", "quantity": quantidade_numerica, "unit": "unidade", "protein": gramas_proteina, "carbs": gramas_carboidratos, "fat": gramas_gordura, "calories": calorias}
  ],
  "totals": {"protein": total_proteina, "carbs": total_carboidratos, "fat": total_gordura, "calories": total_calorias}
}

Regras:
- Todos os valores nutricionais devem ser numéricos (não strings)
- Use gramas (g) como unidade padrão quando não especificada
- Para alimentos descritos em unidades (ex: "2 ovos"), converta para gramas (2 ovos = ~100g) e aplique o cálculo proporcional
- Estime quantidades razoáveis quando não informadas (ex: "arroz" sem peso = porção típica de 150g)
- Arredonde valores para 1 casa decimal
- O campo "totals" deve ser a soma exata de todos os alimentos
- Na dúvida entre valores, prefira a média conservadora da TACO
- NUNCA retorne valores de 100g quando a quantidade informada for diferente de 100g`

	// Monta o corpo da requisição
	reqBody := openAIChatRequest{
		Model: s.model,
		Messages: []openAIChatMsg{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: description},
		},
		ResponseFormat: openAIRespFormat{Type: "json_object"},
		Temperature:    0,
		Seed:           42,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Erro ao serializar requisição OpenAI: %v", err)
		return nil, errors.New("erro ao analisar alimentos. Tente novamente em alguns instantes")
	}

	// Cria e envia a requisição HTTP
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Erro ao criar requisição HTTP para OpenAI: %v", err)
		return nil, errors.New("erro ao analisar alimentos. Tente novamente em alguns instantes")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("Erro ao chamar API da OpenAI: %v", err)
		return nil, errors.New("erro ao analisar alimentos. Tente novamente em alguns instantes")
	}
	defer resp.Body.Close()

	// Lê o corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Erro ao ler resposta da OpenAI: %v", err)
		return nil, errors.New("erro ao analisar alimentos. Tente novamente em alguns instantes")
	}

	// Verifica se a API retornou erro HTTP
	if resp.StatusCode != http.StatusOK {
		// Loga o erro detalhado no servidor para depuração, mas NÃO retorna detalhes ao cliente
		// para evitar vazamento de informações sensíveis (chaves, mensagens internas da API)
		var chatResp openAIChatResponse
		if json.Unmarshal(body, &chatResp) == nil && chatResp.Error != nil {
			log.Printf("Erro da API OpenAI (status %d): %s", resp.StatusCode, chatResp.Error.Message)
		} else {
			log.Printf("Erro da API OpenAI (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, errors.New("erro ao analisar alimentos. Tente novamente em alguns instantes")
	}

	// Faz o parse da resposta da OpenAI
	var chatResp openAIChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		log.Printf("Erro ao decodificar resposta da OpenAI: %v", err)
		return nil, errors.New("erro ao analisar alimentos. Tente novamente em alguns instantes")
	}

	// Verifica se há pelo menos uma choice na resposta
	if len(chatResp.Choices) == 0 {
		return nil, errors.New("resposta da OpenAI não contém dados")
	}

	// Extrai o conteúdo JSON da mensagem de resposta
	content := chatResp.Choices[0].Message.Content
	if content == "" {
		return nil, errors.New("resposta da OpenAI está vazia")
	}

	// Faz o parse do JSON retornado pela OpenAI para a estrutura esperada
	var foodResponse model.OpenAIFoodResponse
	if err := json.Unmarshal([]byte(content), &foodResponse); err != nil {
		log.Printf("Erro ao decodificar JSON de alimentos da OpenAI: %v", err)
		return nil, errors.New("erro ao analisar alimentos. Tente novamente em alguns instantes")
	}

	// Valida que a resposta contém pelo menos um alimento
	if len(foodResponse.Foods) == 0 {
		return nil, errors.New("a OpenAI não identificou nenhum alimento na descrição")
	}

	return &foodResponse, nil
}
