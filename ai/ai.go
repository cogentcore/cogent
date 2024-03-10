package main

import (
	"bufio"
	"github.com/aandrew-me/tgpt/v2/structs"
	"strings"
	"time"

	"cogentcore.org/core/coredom"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/xe"
)

func main() {
	b := gi.NewBody("Cogent AI")
	b.AddAppBar(func(tb *gi.Toolbar) {
		gi.NewButton(tb).SetText("Install") //todo set icon and merge ollama doc md files into s dom tree view
		gi.NewButton(tb).SetText("Start server").OnClick(func(e events.Event) {
			xe.Run("ollama", "serve")
		})
		gi.NewButton(tb).SetText("Stop server").OnClick(func(e events.Event) {
			//todo kill thread ?
			//netstat -aon|findstr 11434
		})
		gi.NewButton(tb).SetText("Logs")
		gi.NewButton(tb).SetText("About").SetIcon(icons.Info)
	})

	splits := gi.NewSplits(b)

	leftFrame := gi.NewFrame(splits)
	leftFrame.Style(func(s *styles.Style) { s.Direction = styles.Column })

	giv.NewTableView(leftFrame).SetSlice(&Models).SetReadOnly(true)

	newFrame := gi.NewFrame(leftFrame)
	newFrame.Style(func(s *styles.Style) {
		s.Direction = styles.Row
	})
	gi.NewButton(newFrame).SetText("Update all module").OnClick(func(e events.Event) {
		queryModelList()
	}).Style(func(s *styles.Style) {
		s.Align.Self = styles.End
		//s.Min.Set(units.Dp(33))
	})

	message := ""
	gi.NewButton(newFrame).SetText("Run selected module").OnClick(func(e events.Event) {
		if message != "" {
			resp, err := NewRequest("go1.22 Generic type constraints", structs.Params{
				ApiModel:    "gemma:2b",
				ApiKey:      "",
				Provider:    "",
				Temperature: "",
				Top_p:       "",
				Max_length:  "1111111",
				Preprompt:   "",
				ThreadID:    "",
			}, "")
			if err != nil {
				return
			}
			//if !mylog.Error(err) {
			//	return
			//}
			//ss := stream.New("")
			scanner := bufio.NewScanner(resp.Body)

			token := make([]string, 0)
			// Handling each part
			previousText := ""
			for scanner.Scan() { //感觉没有 ollama 实现的快，研究一下
				newText := GetMainText(scanner.Text())
				if len(newText) < 1 {
					continue
				}
				mainText := strings.Replace(newText, previousText, "", -1)
				previousText = newText
				println(mainText)
				//codeText.Print(mainText) //todo 颜色不生效
				//ss.WriteString(mainText)
				token = append(token, mainText)
			}
			//mylog.Error(scanner.Err())
		}

	}).Style(func(s *styles.Style) {
		s.Align.Self = styles.End
		//s.Min.Set(units.Dp(33))
	})

	rightSplits := gi.NewSplits(splits)
	splits.SetSplits(.2, .8)

	frame := gi.NewFrame(rightSplits)
	frame.Style(func(s *styles.Style) { s.Direction = styles.Column })

	answer := gi.NewFrame(frame)
	answer.Style(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
	})
	answer.OnShow(func(e events.Event) {
		go func() {
			total := ""
			for _, token := range tokens { //todo this need replace by "Run selected module").OnClick event
				answer.AsyncLock()
				answer.DeleteChildren()
				total += token
				grr.Log(coredom.ReadMDString(coredom.NewContext(), answer, total))
				answer.Update()
				answer.AsyncUnlock()
				time.Sleep(100 * time.Millisecond)
			}
		}()
	})

	prompt := gi.NewFrame(frame)
	prompt.Style(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Align.Items = styles.Center
	})
	gi.NewButton(prompt).SetIcon(icons.Add)
	textField := gi.NewTextField(prompt).SetType(gi.TextFieldOutlined).SetPlaceholder("Enter a prompt here")
	textField.Style(func(s *styles.Style) {
		s.Max.X.Zero()
	})
	textField.OnInput(func(e events.Event) {
		message = textField.Text()
	})

	//newFrame := gi.NewFrame(downFrame)
	//newFrame.Style(func(s *styles.Style) {
	//	s.Direction = styles.Column
	//	s.Align.Self = styles.End
	//})

	gi.NewButton(prompt).SetIcon(icons.Send)

	rightSplits.SetSplits(.6, .4)

	b.RunMainWindow()
}

func queryModelList() {
	//res, err := http.Get("https://ollama.com/library")
	//if !mylog.Error(err) {
	//	return
	//}
	//defer res.Body.Close()
	//if res.StatusCode != 200 {
	//	log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	//}
	//doc, err := goquery.NewDocumentFromReader(res.Body) //todo get tags and make a treeView
	//if !mylog.Error(err) {
	//	return
	//}
	//Models := make([]Model, 0)
	//doc.Find("a").Each(func(i int, s *goquery.Selection) {
	//	title := s.Find("h2").Text()
	//
	//	if title == "" {
	//		return
	//	}
	//	title = unescape(title)
	//	description := s.Find("p").First().Text()
	//	Models = append(Models, Model{
	//		title:       title,
	//		description: description,
	//	})
	//})
	//mylog.Struct(Models)
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

type Model struct {
	Title       string
	Description string
}

var (
	Models = []Model{{Title: "gemma", Description: "Gemma is a family of lightweight, state-of-the-art open models built by Google DeepMind."}, {Title: "llama2", Description: "Llama 2 is a collection of foundation language models ranging from 7B to 70B parameters."}, {Title: "mistral", Description: "The 7B model released by Mistral AI, updated to version 0.2."}, {Title: "mixtral", Description: "A high-quality Mixture of Experts (MoE) model with open weights by Mistral AI."}, {Title: "llava", Description: "� LLaVA is a novel end-to-end trained large multimodal m odel that combines a vision encoder and Vicuna for general-purpose visual and language understanding. Updated to version 1.6."}, {Title: "neural-chat", Description: "A fine-tuned model based on Mistral with good coverage of domain and language."}, {Title: "codellama", Description: "A large language model that can use text prompts to generate and discuss code."}, {Title: "dolphin-mixtral", Description: "An uncensored, fine-tuned model based on the Mixtral mixture of experts model that excels at coding tasks. Created by Eric Hartford."}, {Title: "mistral-openorca", Description: "Mistral OpenOrca is a 7 billion parameter model, fine-tuned on top of the Mistral 7B model using the OpenOrca dataset."}, {Title: "qwen", Description: "Qwen 1.5 is a series of large language models by Alibaba Cloud spanning from 0.5B to 72B parameters"}, {Title: "llama2-uncensored", Description: "Uncensored Llama 2 model by George Sung and Jarrad Hope."}, {Title: "nous-hermes2", Description: "The powerful family of models by Nous Research that excels at scientific discussion and coding tasks."}, {Title: "phi", Description: "Phi-2: a 2.7B language model by Microsoft Research that demonstrates outstanding reasoning and language understanding capabilities."}, {Title: "deepseek-coder", Description: "DeepSeek Coder is a capable coding model trained on two trillion code and natural language tokens."}, {Title: "orca-mini", Description: "A general-purpose model ranging from 3 billion parameters to 70 billion, suitable for entry-level hardware."}, {Title: "dolphin-mistral", Description: "The uncensored Dolphin model based on Mistral that excels at coding tasks. Updated to version 2.6."}, {Title: "wizard-vicuna-uncensored", Description: "Wizard Vicuna Uncensored is a 7B, 13B, and 30B parameter model based on Llama 2 uncensored by Eric Hartford."}, {Title: "vicuna", Description: "General use chat model based on Llama and Llama 2 with 2K to 16K context sizes."}, {Title: "zephyr", Description: "Zephyr beta is a fine-tuned 7B version of mistral that was trained on on a mix of publicly available, synthetic datasets."}, {Title: "openhermes", Description: "OpenHermes 2.5 is a 7B model fine-tuned by Teknium on Mistral with fully open datasets."}, {Title: "llama2-chinese", Description: "Llama 2 based model fine tuned to improve Chinese dialogue ability."}, {Title: "wizardcoder", Description: "State-of-the-art code generation model"}, {Title: "tinyllama", Description: "The TinyLlama project is an open endeavor to train a compact 1.1B Llama model on 3 trillion tokens."}, {Title: "openchat", Description: "A family of open-source models trained on a wide variety of data, surpassing ChatGPT on various benchmarks. Updated to version 3.5-0106."}, {Title: "phind-codellama", Description: "Code generation model based on Code Llama."}, {Title: "tinydolphin", Description: "An experimental 1.1B parameter model trained on the new Dolphin 2.8 dataset by Eric Hartford and based on TinyLlama."}, {Title: "orca2", Description: "Orca 2 is built by Microsoft research, and are a fine-tuned version of Meta's Llama 2 models.  The model is designed to excel particularly in reasoning."}, {Title: "falcon", Description: "A large language model built by the Technology Innovation Institute (TII) for use in summarization, text generation, and chat bots."}, {Title: "wizard-math", Description: "Model focused on math and logic problems"}, {Title: "yi", Description: "A high-performing, bilingual language model."}, {Title: "starcoder", Description: "StarCoder is a code generation model trained on 80+ programming languages."}, {Title: "dolphin-phi", Description: "2.7B uncensored Dolphin model by Eric Hartford, based on the Phi language model by Microsoft Research."}, {Title: "nous-hermes", Description: "General use models based on Llama and Llama 2 from Nous Research."}, {Title: "starling-lm", Description: "Starling is a large language model trained by reinforcement learning from AI feedback focused on improving chatbot helpfulness."}, {Title: "stable-code", Description: "Stable Code 3B is a model offering accurate and responsive code completion at a level on par with models such as CodeLLaMA 7B that are 2.5x larger."}, {Title: "codeup", Description: "Great code generation model based on Llama2."}, {Title: "medllama2", Description: "Fine-tuned Llama 2 model to answer medical questions based on an open source medical dataset. "}, {Title: "bakllava", Description: "BakLLaVA is a multimodal model consisting of the Mistral 7B base model augmented with the LLaVA  architecture."}, {Title: "wizardlm-uncensored", Description: "Uncensored version of Wizard LM model "}, {Title: "everythinglm", Description: "Uncensored Llama2 based model with support for a 16K context window."}, {Title: "solar", Description: "A compact, yet powerful 10.7B large language model designed for single-turn conversation."}, {Title: "starcoder2", Description: "StarCoder2 is the next generation of transparently trained open code LLMs that comes in three sizes: 3B, 7B and 15B parameters. "}, {Title: "nomic-embed-text", Description: "A high-performing open embedding model with a large token context window."}, {Title: "stable-beluga", Description: "Llama 2 based model fine tuned on an Orca-style dataset. Originally called Free Willy."}, {Title: "sqlcoder", Description: "SQLCoder is a code completion model fined-tuned on StarCoder for SQL generation tasks"}, {Title: "nous-hermes2-mixtral", Description: "The Nous Hermes 2 model from Nous Research, now trained over Mixtral."}, {Title: "yarn-mistral", Description: "An extension of Mistral to support context windows of 64K or 128K."}, {Title: "samantha-mistral", Description: "A companion assistant trained in philosophy, psychology, and personal relationships. Based on Mistral."}, {Title: "meditron", Description: "Open-source medical large language model adapted from Llama 2 to the medical do"}, {Title: "stablelm2", Description: "Stable LM 2 1.6B is a state-of-the-art 1.6 billion parameter small language model trained on multilingual data in English, Spanish, German, Italian, French, Portuguese, and Dutch."}, {Title: "stablelm-zephyr", Description: "A lightweight chat model allowing accurate, and responsive output without requiring high-end hardware."}, {Title: "magicoder", Description: "� Magicoder is a family of 7B parameter models trained on 75K synthetic  instruction data using OSS-Instruct, a novel approach to enlightening LLMs with open-source code snippets."}, {Title: "wizard-vicuna", Description: "Wizard Vicuna is a 13B parameter model based on Llama 2 trained by MelodysDreamj."}, {Title: "yarn-llama2", Description: "An extension of Llama 2 that supports a context of up to 128k tokens."}, {Title: "deepseek-llm", Description: "An advanced language model crafted with 2 trillion bilingual tokens."}, {Title: "llama-pro", Description: "An expansion of Llama 2 that specializes in integrating both general language understanding and domain-specific knowledge, particularly in programming and mathematics."}, {Title: "mistrallite", Description: "MistralLite is a fine-tuned model based on Mistral with enhanced capabilities of processing long contexts."}, {Title: "codebooga", Description: "A high-performing code instruct model created by merging two existing code models."}, {Title: "open-orca-platypus2", Description: "Merge of the Open Orca OpenChat model and the Garage-bAInd Platypus 2 model. Designed for chat and code generation."}, {Title: "nexusraven", Description: "Nexus Raven is a 13B instruction tuned model for function calling tasks. "}, {Title: "goliath", Description: "A language model created by combining two fine-tuned Llama 2 70B models into one."}, {Title: "notux", Description: "A top-performing mixture of experts model, fine-tuned with high-quality data."}, {Title: "alfred", Description: "A robust conversational model designed to be used for both chat and instruct use cases."}, {Title: "megadolphin", Description: "MegaDolphin-2.2-120b is a transformation of Dolphin-2.2-70b created by interleaving the model with itself."}, {Title: "xwinlm", Description: "Conversational model based on Llama 2 that performs competitively on various benchmarks."}, {Title: "wizardlm", Description: "General use 70 billion parameter model based on Llama 2."}, {Title: "notus", Description: "A 7B chat model fine-tuned with high-quality data and based on Zephyr."}, {Title: "duckdb-nsql", Description: "7B parameter text-to-SQL model made by MotherDuck and Numbers Station."}, {Title: "all-minilm", Description: "Embedding models on very large sentence level datasets."}, {Title: "dolphincoder", Description: "An uncensored variant of the Dolphin model family that excels at coding, based on StarCoder2."}}
	tokens = []string{"**", "Generic", " type", " constraints", "**", " allow", " you", " to", " specify", " constraints", " on", " a", " type", " that", " can", " vary", " depending", " on", " the", " specific", " type", " being", " instantiated", ".", "\n\n", "**", "Syntax", ":**", "\n\n", "```", "go", "\n", "type", " Name", "[", "T", " any", "]", " string", "\n", "```", "\n\n", "**", "Parameters", ":**", "\n\n", "*", " `", "T", "`:", " The", " type", " variable", ".", " It", " can", " be", " any", " type", ",", " including", " primitive", " types", ",", " structures", ",", " and", " functions", ".", "\n\n", "**", "Examples", ":**", "\n\n", "*", " ", "Integer", " constraint", ":**", "\n", "```", "go", "\n", "type", " Age", "[", "T", " int", "]", " int", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " an", " integer", " type", ".", "\n\n", "*", " ", "String", " constraint", ":**", "\n", "```", "go", "\n", "type", " Name", "[", "T", " string", "]", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " a", " string", " type", ".", "\n\n", "*", " ", "Struct", " constraint", ":**", "\n", "```", "go", "\n", "type", " User", "[", "T", " struct", "]", " {", "\n", "  ", "Name", " string", "\n", "  ", "Age", "  ", "int", "\n", "}", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " a", " struct", " type", " with", " at", " least", " two", " fields", " named", " `", "Name", "`", " and", " `", "Age", "`.", "\n\n", "*", " ", "Function", " constraint", ":**", "\n", "```", "go", "\n", "type", " Calculator", "[", "T", " any", "]", " func", "(", "T", ",", " T", ")", " T", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " a", " type", " that", " implements", " the", " `", "Calculator", "`", " interface", ".", "\n\n", "**", "Benefits", " of", " using", " generic", " type", " constraints", ":**", "\n\n", "*", " ", "Code", " reus", "ability", ":**", " You", " can", " apply", " the", " same", " constraint", " to", " multiple", " types", ",", " reducing", " code", " duplication", ".", "\n", "*", " ", "Type", " safety", ":**", " Constraints", " ensure", " that", " only", " valid", " types", " are", " used", ",", " preventing", " runtime", " errors", ".", "\n", "*", " ", "Improved", " maintain", "ability", ":**", " By", " separating", " the", " constraint", " from", " the", " type", ",", " it", " becomes", " easier", " to", " understand", " and", " modify", ".", "\n\n", "**", "Note", ":**", "\n\n", "*", " Generic", " type", " constraints", " are", " not", " applicable", " to", " primitive", " types", " (", "e", ".", "g", ".,", " `", "int", "`,", " `", "string", "`).", "\n", "*", " Constraints", " can", " be", " applied", " to", " function", " types", " only", " if", " the", " function", " is", " generic", ".", "\n", "*", " Constraints", " can", " be", " used", " with", " type", " parameters", ",", " allowing", " you", " to", " specify", " different", " constraints", " for", " different", " types", "."}
)
