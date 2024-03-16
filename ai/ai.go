package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"cogentcore.org/core/coredom"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/xe"

	"github.com/PuerkitoBio/goquery"
	"github.com/aandrew-me/tgpt/v2/structs"
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/stream"
	"github.com/goradd/maps"

	"cogentcore.org/cogent/ai/pkg/tree"
)

//go:generate go install .

func main() {
	b := gi.NewBody("Cogent AI")
	b.AddAppBar(func(tb *gi.Toolbar) {
		gi.NewButton(tb).SetText("Install") // todo set icon and merge ollama doc md files into s dom tree view
		gi.NewButton(tb).SetText("Start server").OnClick(func(e events.Event) {
			gi.ErrorSnackbar(b, xe.Verbose().Run("ollama", "serve"))
		})
		gi.NewButton(tb).SetText("Stop server").OnClick(func(e events.Event) {
			// todo kill thread ?
			// netstat -aon|findstr 11434
		})
		gi.NewButton(tb).SetText("Logs")                      // todo add a new windows show log and set ico
		gi.NewButton(tb).SetText("About").SetIcon(icons.Info) // todo add a new windows show some info
	})

	splits := gi.NewSplits(b)

	leftFrame := gi.NewFrame(splits)
	leftFrame.Style(func(s *styles.Style) { s.Direction = styles.Column })

	tableView := giv.NewTableView(leftFrame).SetSlice(&Models)
	tableView.SetReadOnly(true)

	newFrame := gi.NewFrame(leftFrame)
	newFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	gi.NewButton(newFrame).SetText("Update module list").OnClick(func(e events.Event) {
		QueryModelList()
	})

	gi.NewButton(newFrame).SetText("Run selected module").OnClick(func(e events.Event) {
		// model := Models[tableView.SelIdx]
		// cmd.RunArgs("ollama", "run", model.Name)//not need
	})
	gi.NewButton(newFrame).SetText("Stop selected module").OnClick(func(e events.Event) {
		// model := Models[tableView.SelIdx]
		// cmd.RunArgs("ollama", "stop",model.Name)//not need
	})

	rightSplits := gi.NewSplits(splits)
	splits.SetSplits(.2, .8)

	frame := gi.NewFrame(rightSplits)
	frame.Style(func(s *styles.Style) { s.Direction = styles.Column })

	answer := gi.NewFrame(frame)
	answer.Style(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
	})

	prompt := gi.NewFrame(frame)
	prompt.Style(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Align.Items = styles.Center
	})
	gi.NewButton(prompt).SetIcon(icons.Add)
	textField := gi.NewTextField(prompt).SetType(gi.TextFieldOutlined).SetPlaceholder("Enter a prompt here")
	textField.Style(func(s *styles.Style) { s.Max.X.Zero() })

	gi.NewButton(prompt).SetIcon(icons.Send).OnClick(func(e events.Event) {
		if textField.Text() == "" {
			mylog.Error("textField.Text() == \"\"")
			return
		}
		go func() {
			mylog.Warning("connect serve", "Send "+strconv.Quote(textField.Text())+" to the serve,please wait a while")
			// model := Models[tableView.SelIdx]
			resp, err := NewRequest(textField.Text(), structs.Params{ // go1.22 Generic type constraints
				// ApiModel: model.Name,
				ApiModel:    "gemma:2b",
				ApiKey:      "",
				Provider:    "",
				Temperature: "",
				Top_p:       "",
				Max_length:  "1111111",
				Preprompt:   "",
				ThreadID:    "",
			}, "")
			if !mylog.Error(err) { // todo  timeout ? need set it
				return
			}
			scanner := bufio.NewScanner(resp.Body)
			allToken := ""
			for scanner.Scan() {
				token := HandleToken(scanner.Text())
				if token == "" {
					continue
				}
				print(token)

				answer.AsyncLock()
				answer.DeleteChildren() //todo can not save chat history

				//reset answer style
				answer.Style(func(s *styles.Style) {
					s.Direction = styles.Column
				})

				//  need save chat list layout for show chat history
				you := gi.NewFrame(answer)
				you.Style(func(s *styles.Style) {
					s.Direction = styles.Row
				})
				gi.NewLabel(you).SetText("yuo:").Style(func(s *styles.Style) { //todo NewLabel seems can not set svg icon
					s.Align.Self = styles.Start
				})
				youSend := gi.NewTextField(you).SetType(gi.TextFieldOutlined) //todo need more type
				youSend.SetText(textField.Text())                             //todo if we send code block or md need highlight it
				youSend.Style(func(s *styles.Style) {
					s.Align.Self = styles.End
				})
				//todo need support emoji  😅 😁 😍 🤥 https://www.emojiall.com/zh-hans/categories/A

				ai := gi.NewFrame(answer)
				ai.Style(func(s *styles.Style) {
					s.Direction = styles.Row
				})
				gi.NewLabel(ai).SetText("ai:").Style(func(s *styles.Style) {
					s.Align.Self = styles.Start
				})

				//now need given ReadMDString a NewFrame? and set s.Align.Self = styles.End ?
				mdFrame := gi.NewFrame(answer) //todo rename answer as chatPair
				mdFrame.Style(func(s *styles.Style) {
					s.Align.Self = styles.End
				})

				allToken += token

				if !mylog.Error(coredom.ReadMDString(coredom.NewContext(), mdFrame, allToken)) {
					return
				}
				answer.Update()
				answer.AsyncUnlock()
			}
			mylog.Error(scanner.Err())
			// mylog.Error(resp.Body.Close()) //not do,unknown reason
		}()
	})

	rightSplits.SetSplits(.6, .4)

	b.RunMainWindow()
}

func QueryModelList() {
	resp, err := http.Get("https://ollama.com/library")
	if !mylog.Error(err) {
		return
	}
	defer func() { mylog.Error(resp.Body.Close()) }()
	if resp.StatusCode != 200 {
		mylog.Error(fmt.Sprintf("status code error: %d %s", resp.StatusCode, resp.Status))
		return
	}
	root := queryModelList(resp.Body)
	root.WalkContainer(func(node *tree.Node[Model]) {
		QueryModelTags(node.Data.Name, node) //node is every container of model node
	})
	//todo last need save to json file when the test passed
}

func queryModelList(r io.Reader) (root *tree.Node[Model]) {
	root = tree.NewNode("aiModel", true, Model{
		Name:        "root",
		Description: "",
		UpdateTime:  "",
		Hash:        "",
		Size:        "",
		Children:    nil,
	})
	doc, err := goquery.NewDocumentFromReader(r)
	if !mylog.Error(err) {
		return
	}
	doc.Find("a.group").Each(func(i int, s *goquery.Selection) {
		name := s.Find("h2.mb-3").Text()
		name = unescape(name)
		description := s.Find("p.mb-4").First().Text()
		model := Model{
			Name:        name,
			Description: description,
			UpdateTime:  "",
			Hash:        "",
			Size:        "",
			Children:    make([]Model, 0),
		}
		parent := tree.NewNode(name, false, model)
		root.AddChild(parent)
	})
	return
}

func QueryModelTags(name string, parent *tree.Node[Model]) {
	resp, err := http.Get("https://ollama.com/library/" + name + "/tags")
	if !mylog.Error(err) {
		return
	}
	defer func() { mylog.Error(resp.Body.Close()) }()
	queryModelTags(resp.Body, parent)
}

func queryModelTags(r io.Reader, parent *tree.Node[Model]) {
	doc, err := goquery.NewDocumentFromReader(r)
	if !mylog.Error(err) {
		return
	}
	doc.Find("a.group").Each(func(i int, s *goquery.Selection) {
		//tag := s.Find(".break-all").Text() //not need
		modelWithTag := ""
		fnFindModelName := func() {
			href, exists := s.Attr("href")
			if exists {
				// https://ollama.com/library/llama2:latest
				// /library/gemma:latest
				_, after, found := strings.Cut(href, "/library/")
				if !found {
					return
				}
				modelWithTag = after

			}
			if modelWithTag == "" {
				mylog.Error("not find model name in tags")
				return
			}
		}

		fnFindModelName()

		modelInfo := s.Find("span").Text()
		lines, ok := stream.New(modelInfo).ToLines()
		if !ok {
			mylog.Error("modelInfo ToLines not ok")
			return
		}
		modelInfoSplit := strings.Split(lines[1], " • ")
		if strings.Contains(modelWithTag, parent.Data.Name) {
			model := Model{
				//Name: parent.Data.Name + ":" + tag,
				Name:        modelWithTag,
				Description: parent.Data.Description,
				UpdateTime:  strings.TrimSpace(lines[2]),
				Hash:        strings.TrimSpace(modelInfoSplit[0]),
				Size:        modelInfoSplit[1],
				Children:    nil,
				Node:        nil,
			}
			model.Node = tree.NewNode[*Model]("", false, &model)
			ModelMap.Set(modelWithTag, model)

			mylog.Struct(model)

			c := tree.NewNode(modelWithTag, false, model)
			c = c //todo debug
			parent.AddChild(tree.NewNode(modelWithTag, false, model))
		}
	})
}

func unescape(s string) string {
	return strings.NewReplacer(
		`\n`, "",
		"\n", "",
		` `, "",
		//`\\`, "",
		//`\n`, "\n",
		//`\r`, "\r",
		//`\t`, "\t",
		//`\"`, `"`,
		//`\u003e`, `>`,
		//`\u003c`, `<`,
		//`\u0026`, `&`,
	).Replace(s)
}

type (
	Model struct {
		Name               string
		Description        string
		UpdateTime         string
		Hash               string
		Size               string
		Children           []Model `json:"_"` //tags
		*tree.Node[*Model] `json:"_"`
	}
)

func resetModels() {
	Models = Models[:0]
}

var ModelMap = new(maps.Map[string, Model])
var Models = []Model{{Name: "gemma", Description: "Gemma is a family of lightweight, state-of-the-art open models built by Google DeepMind."}, {Name: "llama2", Description: "Llama 2 is a collection of foundation language models ranging from 7B to 70B parameters."}, {Name: "mistral", Description: "The 7B model released by Mistral AI, updated to version 0.2."}, {Name: "mixtral", Description: "A high-quality Mixture of Experts (MoE) model with open weights by Mistral AI."}, {Name: "llava", Description: "� LLaVA is a novel end-to-end trained large multimodal m odel that combines a vision encoder and Vicuna for general-purpose visual and language understanding. Updated to version 1.6."}, {Name: "neural-chat", Description: "A fine-tuned model based on Mistral with good coverage of domain and language."}, {Name: "codellama", Description: "A large language model that can use text prompts to generate and discuss code."}, {Name: "dolphin-mixtral", Description: "An uncensored, fine-tuned model based on the Mixtral mixture of experts model that excels at coding tasks. Created by Eric Hartford."}, {Name: "mistral-openorca", Description: "Mistral OpenOrca is a 7 billion parameter model, fine-tuned on top of the Mistral 7B model using the OpenOrca dataset."}, {Name: "qwen", Description: "Qwen 1.5 is a series of large language models by Alibaba Cloud spanning from 0.5B to 72B parameters"}, {Name: "llama2-uncensored", Description: "Uncensored Llama 2 model by George Sung and Jarrad Hope."}, {Name: "nous-hermes2", Description: "The powerful family of models by Nous Research that excels at scientific discussion and coding tasks."}, {Name: "phi", Description: "Phi-2: a 2.7B language model by Microsoft Research that demonstrates outstanding reasoning and language understanding capabilities."}, {Name: "deepseek-coder", Description: "DeepSeek Coder is a capable coding model trained on two trillion code and natural language tokens."}, {Name: "orca-mini", Description: "A general-purpose model ranging from 3 billion parameters to 70 billion, suitable for entry-level hardware."}, {Name: "dolphin-mistral", Description: "The uncensored Dolphin model based on Mistral that excels at coding tasks. Updated to version 2.6."}, {Name: "wizard-vicuna-uncensored", Description: "Wizard Vicuna Uncensored is a 7B, 13B, and 30B parameter model based on Llama 2 uncensored by Eric Hartford."}, {Name: "vicuna", Description: "General use chat model based on Llama and Llama 2 with 2K to 16K context sizes."}, {Name: "zephyr", Description: "Zephyr beta is a fine-tuned 7B version of mistral that was trained on on a mix of publicly available, synthetic datasets."}, {Name: "openhermes", Description: "OpenHermes 2.5 is a 7B model fine-tuned by Teknium on Mistral with fully open datasets."}, {Name: "llama2-chinese", Description: "Llama 2 based model fine tuned to improve Chinese dialogue ability."}, {Name: "wizardcoder", Description: "State-of-the-art code generation model"}, {Name: "tinyllama", Description: "The TinyLlama project is an open endeavor to train a compact 1.1B Llama model on 3 trillion tokens."}, {Name: "openchat", Description: "A family of open-source models trained on a wide variety of data, surpassing ChatGPT on various benchmarks. Updated to version 3.5-0106."}, {Name: "phind-codellama", Description: "Code generation model based on Code Llama."}, {Name: "tinydolphin", Description: "An experimental 1.1B parameter model trained on the new Dolphin 2.8 dataset by Eric Hartford and based on TinyLlama."}, {Name: "orca2", Description: "Orca 2 is built by Microsoft research, and are a fine-tuned version of Meta's Llama 2 models.  The model is designed to excel particularly in reasoning."}, {Name: "falcon", Description: "A large language model built by the Technology Innovation Institute (TII) for use in summarization, text generation, and chat bots."}, {Name: "wizard-math", Description: "Model focused on math and logic problems"}, {Name: "yi", Description: "A high-performing, bilingual language model."}, {Name: "starcoder", Description: "StarCoder is a code generation model trained on 80+ programming languages."}, {Name: "dolphin-phi", Description: "2.7B uncensored Dolphin model by Eric Hartford, based on the Phi language model by Microsoft Research."}, {Name: "nous-hermes", Description: "General use models based on Llama and Llama 2 from Nous Research."}, {Name: "starling-lm", Description: "Starling is a large language model trained by reinforcement learning from AI feedback focused on improving chatbot helpfulness."}, {Name: "stable-code", Description: "Stable Code 3B is a model offering accurate and responsive code completion at a level on par with models such as CodeLLaMA 7B that are 2.5x larger."}, {Name: "codeup", Description: "Great code generation model based on Llama2."}, {Name: "medllama2", Description: "Fine-tuned Llama 2 model to answer medical questions based on an open source medical dataset. "}, {Name: "bakllava", Description: "BakLLaVA is a multimodal model consisting of the Mistral 7B base model augmented with the LLaVA  architecture."}, {Name: "wizardlm-uncensored", Description: "Uncensored version of Wizard LM model "}, {Name: "everythinglm", Description: "Uncensored Llama2 based model with support for a 16K context window."}, {Name: "solar", Description: "A compact, yet powerful 10.7B large language model designed for single-turn conversation."}, {Name: "starcoder2", Description: "StarCoder2 is the next generation of transparently trained open code LLMs that comes in three sizes: 3B, 7B and 15B parameters. "}, {Name: "nomic-embed-text", Description: "A high-performing open embedding model with a large token context window."}, {Name: "stable-beluga", Description: "Llama 2 based model fine tuned on an Orca-style dataset. Originally called Free Willy."}, {Name: "sqlcoder", Description: "SQLCoder is a code completion model fined-tuned on StarCoder for SQL generation tasks"}, {Name: "nous-hermes2-mixtral", Description: "The Nous Hermes 2 model from Nous Research, now trained over Mixtral."}, {Name: "yarn-mistral", Description: "An extension of Mistral to support context windows of 64K or 128K."}, {Name: "samantha-mistral", Description: "A companion assistant trained in philosophy, psychology, and personal relationships. Based on Mistral."}, {Name: "meditron", Description: "Open-source medical large language model adapted from Llama 2 to the medical do"}, {Name: "stablelm2", Description: "Stable LM 2 1.6B is a state-of-the-art 1.6 billion parameter small language model trained on multilingual data in English, Spanish, German, Italian, French, Portuguese, and Dutch."}, {Name: "stablelm-zephyr", Description: "A lightweight chat model allowing accurate, and responsive output without requiring high-end hardware."}, {Name: "magicoder", Description: "� Magicoder is a family of 7B parameter models trained on 75K synthetic  instruction data using OSS-Instruct, a novel approach to enlightening LLMs with open-source code snippets."}, {Name: "wizard-vicuna", Description: "Wizard Vicuna is a 13B parameter model based on Llama 2 trained by MelodysDreamj."}, {Name: "yarn-llama2", Description: "An extension of Llama 2 that supports a context of up to 128k tokens."}, {Name: "deepseek-llm", Description: "An advanced language model crafted with 2 trillion bilingual tokens."}, {Name: "llama-pro", Description: "An expansion of Llama 2 that specializes in integrating both general language understanding and domain-specific knowledge, particularly in programming and mathematics."}, {Name: "mistrallite", Description: "MistralLite is a fine-tuned model based on Mistral with enhanced capabilities of processing long contexts."}, {Name: "codebooga", Description: "A high-performing code instruct model created by merging two existing code models."}, {Name: "open-orca-platypus2", Description: "Merge of the Open Orca OpenChat model and the Garage-bAInd Platypus 2 model. Designed for chat and code generation."}, {Name: "nexusraven", Description: "Nexus Raven is a 13B instruction tuned model for function calling tasks. "}, {Name: "goliath", Description: "A language model created by combining two fine-tuned Llama 2 70B models into one."}, {Name: "notux", Description: "A top-performing mixture of experts model, fine-tuned with high-quality data."}, {Name: "alfred", Description: "A robust conversational model designed to be used for both chat and instruct use cases."}, {Name: "megadolphin", Description: "MegaDolphin-2.2-120b is a transformation of Dolphin-2.2-70b created by interleaving the model with itself."}, {Name: "xwinlm", Description: "Conversational model based on Llama 2 that performs competitively on various benchmarks."}, {Name: "wizardlm", Description: "General use 70 billion parameter model based on Llama 2."}, {Name: "notus", Description: "A 7B chat model fine-tuned with high-quality data and based on Zephyr."}, {Name: "duckdb-nsql", Description: "7B parameter text-to-SQL model made by MotherDuck and Numbers Station."}, {Name: "all-minilm", Description: "Embedding models on very large sentence level datasets."}, {Name: "dolphincoder", Description: "An uncensored variant of the Dolphin model family that excels at coding, based on StarCoder2."}}
