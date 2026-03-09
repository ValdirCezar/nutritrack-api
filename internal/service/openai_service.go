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
- Os valores devem refletir médias reais de composição por 100g, ajustados para a quantidade informada.
- NÃO superestime nem subestime valores. Use as médias da tabela de referência.

Exemplos de referência TACO (por unidade média):
- 1 ovo cozido (~50g): ~6.3g proteína, 0.6g carboidrato, 4.2g gordura, 66 kcal
- 100g arroz branco cozido: ~2.5g proteína, 28.1g carboidrato, 0.2g gordura, 128 kcal
- 100g peito de frango grelhado: ~31.5g proteína, 0g carboidrato, 1.3g gordura, 159 kcal
- 100g feijão carioca cozido: ~4.8g proteína, 13.6g carboidrato, 0.5g gordura, 76 kcal
- 1 banana prata média (~86g): ~1.1g proteína, 22g carboidrato, 0.1g gordura, 89 kcal

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
- Para alimentos descritos em unidades (ex: "2 ovos"), calcule baseado no peso médio da unidade
- Estime quantidades razoáveis quando não informadas (ex: "arroz" sem peso = porção típica de 150g)
- Arredonde valores para 1 casa decimal
- O campo "totals" deve ser a soma exata de todos os alimentos
- Na dúvida entre valores, prefira a média conservadora da TACO`

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
