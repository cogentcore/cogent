package main

import (
	"strings"
	"time"

	"cogentcore.org/core/coredom"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
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
	giv.NewFileView(leftFrame)
	gi.NewButton(leftFrame).SetText("Update module").Style(func(s *styles.Style) {
		s.Align.Self = styles.End
		s.Min.Set(units.Dp(33))
	})
	gi.NewButton(leftFrame).SetText("Run module").OnClick(func(e events.Event) {
		queryModelList()
	}).Style(func(s *styles.Style) {
		s.Align.Self = styles.End
		s.Min.Set(units.Dp(33))
	})

	rightSplits := gi.NewSplits(splits)
	splits.SetSplits(.2, .8)

	frame := gi.NewFrame(rightSplits)
	frame.Style(func(s *styles.Style) { s.Direction = styles.Column })

	answer := gi.NewFrame(frame)
	answer.OnShow(func(e events.Event) {
		go func() {
			total := ""
			for _, token := range tokens {
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

	downFrame := gi.NewFrame(frame)
	downFrame.Style(func(s *styles.Style) { s.Direction = styles.Row })
	topic := gi.NewButton(downFrame).SetText("New topic").SetIcon(icons.ClearAll)
	topic.Style(func(s *styles.Style) {
		//s.Min.Set(units.Dp(33))
	})
	gi.NewTextField(downFrame).SetType(gi.TextFieldOutlined).SetPlaceholder("Enter a prompt here").Style(func(s *styles.Style) {
		s.Max.X.Zero()
	})

	//newFrame := gi.NewFrame(downFrame)
	//newFrame.Style(func(s *styles.Style) {
	//	s.Direction = styles.Column
	//	s.Align.Self = styles.End
	//})

	gi.NewButton(downFrame).SetText("Send").Style(func(s *styles.Style) {
		//s.Min.Set(units.Dp(33))
	})

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
	title       string
	description string
}

var (
	Models = []Model{{title: "gemma", description: "Gemma is a family of lightweight, state-of-the-art open models built by Google DeepMind."}, {title: "llama2", description: "Llama 2 is a collection of foundation language models ranging from 7B to 70B parameters."}, {title: "mistral", description: "The 7B model released by Mistral AI, updated to version 0.2."}, {title: "mixtral", description: "A high-quality Mixture of Experts (MoE) model with open weights by Mistral AI."}, {title: "llava", description: "� LLaVA is a novel end-to-end trained large multimodal m odel that combines a vision encoder and Vicuna for general-purpose visual and language understanding. Updated to version 1.6."}, {title: "neural-chat", description: "A fine-tuned model based on Mistral with good coverage of domain and language."}, {title: "codellama", description: "A large language model that can use text prompts to generate and discuss code."}, {title: "dolphin-mixtral", description: "An uncensored, fine-tuned model based on the Mixtral mixture of experts model that excels at coding tasks. Created by Eric Hartford."}, {title: "mistral-openorca", description: "Mistral OpenOrca is a 7 billion parameter model, fine-tuned on top of the Mistral 7B model using the OpenOrca dataset."}, {title: "qwen", description: "Qwen 1.5 is a series of large language models by Alibaba Cloud spanning from 0.5B to 72B parameters"}, {title: "llama2-uncensored", description: "Uncensored Llama 2 model by George Sung and Jarrad Hope."}, {title: "nous-hermes2", description: "The powerful family of models by Nous Research that excels at scientific discussion and coding tasks."}, {title: "phi", description: "Phi-2: a 2.7B language model by Microsoft Research that demonstrates outstanding reasoning and language understanding capabilities."}, {title: "deepseek-coder", description: "DeepSeek Coder is a capable coding model trained on two trillion code and natural language tokens."}, {title: "orca-mini", description: "A general-purpose model ranging from 3 billion parameters to 70 billion, suitable for entry-level hardware."}, {title: "dolphin-mistral", description: "The uncensored Dolphin model based on Mistral that excels at coding tasks. Updated to version 2.6."}, {title: "wizard-vicuna-uncensored", description: "Wizard Vicuna Uncensored is a 7B, 13B, and 30B parameter model based on Llama 2 uncensored by Eric Hartford."}, {title: "vicuna", description: "General use chat model based on Llama and Llama 2 with 2K to 16K context sizes."}, {title: "zephyr", description: "Zephyr beta is a fine-tuned 7B version of mistral that was trained on on a mix of publicly available, synthetic datasets."}, {title: "openhermes", description: "OpenHermes 2.5 is a 7B model fine-tuned by Teknium on Mistral with fully open datasets."}, {title: "llama2-chinese", description: "Llama 2 based model fine tuned to improve Chinese dialogue ability."}, {title: "wizardcoder", description: "State-of-the-art code generation model"}, {title: "tinyllama", description: "The TinyLlama project is an open endeavor to train a compact 1.1B Llama model on 3 trillion tokens."}, {title: "openchat", description: "A family of open-source models trained on a wide variety of data, surpassing ChatGPT on various benchmarks. Updated to version 3.5-0106."}, {title: "phind-codellama", description: "Code generation model based on Code Llama."}, {title: "tinydolphin", description: "An experimental 1.1B parameter model trained on the new Dolphin 2.8 dataset by Eric Hartford and based on TinyLlama."}, {title: "orca2", description: "Orca 2 is built by Microsoft research, and are a fine-tuned version of Meta's Llama 2 models.  The model is designed to excel particularly in reasoning."}, {title: "falcon", description: "A large language model built by the Technology Innovation Institute (TII) for use in summarization, text generation, and chat bots."}, {title: "wizard-math", description: "Model focused on math and logic problems"}, {title: "yi", description: "A high-performing, bilingual language model."}, {title: "starcoder", description: "StarCoder is a code generation model trained on 80+ programming languages."}, {title: "dolphin-phi", description: "2.7B uncensored Dolphin model by Eric Hartford, based on the Phi language model by Microsoft Research."}, {title: "nous-hermes", description: "General use models based on Llama and Llama 2 from Nous Research."}, {title: "starling-lm", description: "Starling is a large language model trained by reinforcement learning from AI feedback focused on improving chatbot helpfulness."}, {title: "stable-code", description: "Stable Code 3B is a model offering accurate and responsive code completion at a level on par with models such as CodeLLaMA 7B that are 2.5x larger."}, {title: "codeup", description: "Great code generation model based on Llama2."}, {title: "medllama2", description: "Fine-tuned Llama 2 model to answer medical questions based on an open source medical dataset. "}, {title: "bakllava", description: "BakLLaVA is a multimodal model consisting of the Mistral 7B base model augmented with the LLaVA  architecture."}, {title: "wizardlm-uncensored", description: "Uncensored version of Wizard LM model "}, {title: "everythinglm", description: "Uncensored Llama2 based model with support for a 16K context window."}, {title: "solar", description: "A compact, yet powerful 10.7B large language model designed for single-turn conversation."}, {title: "starcoder2", description: "StarCoder2 is the next generation of transparently trained open code LLMs that comes in three sizes: 3B, 7B and 15B parameters. "}, {title: "nomic-embed-text", description: "A high-performing open embedding model with a large token context window."}, {title: "stable-beluga", description: "Llama 2 based model fine tuned on an Orca-style dataset. Originally called Free Willy."}, {title: "sqlcoder", description: "SQLCoder is a code completion model fined-tuned on StarCoder for SQL generation tasks"}, {title: "nous-hermes2-mixtral", description: "The Nous Hermes 2 model from Nous Research, now trained over Mixtral."}, {title: "yarn-mistral", description: "An extension of Mistral to support context windows of 64K or 128K."}, {title: "samantha-mistral", description: "A companion assistant trained in philosophy, psychology, and personal relationships. Based on Mistral."}, {title: "meditron", description: "Open-source medical large language model adapted from Llama 2 to the medical do"}, {title: "stablelm2", description: "Stable LM 2 1.6B is a state-of-the-art 1.6 billion parameter small language model trained on multilingual data in English, Spanish, German, Italian, French, Portuguese, and Dutch."}, {title: "stablelm-zephyr", description: "A lightweight chat model allowing accurate, and responsive output without requiring high-end hardware."}, {title: "magicoder", description: "� Magicoder is a family of 7B parameter models trained on 75K synthetic  instruction data using OSS-Instruct, a novel approach to enlightening LLMs with open-source code snippets."}, {title: "wizard-vicuna", description: "Wizard Vicuna is a 13B parameter model based on Llama 2 trained by MelodysDreamj."}, {title: "yarn-llama2", description: "An extension of Llama 2 that supports a context of up to 128k tokens."}, {title: "deepseek-llm", description: "An advanced language model crafted with 2 trillion bilingual tokens."}, {title: "llama-pro", description: "An expansion of Llama 2 that specializes in integrating both general language understanding and domain-specific knowledge, particularly in programming and mathematics."}, {title: "mistrallite", description: "MistralLite is a fine-tuned model based on Mistral with enhanced capabilities of processing long contexts."}, {title: "codebooga", description: "A high-performing code instruct model created by merging two existing code models."}, {title: "open-orca-platypus2", description: "Merge of the Open Orca OpenChat model and the Garage-bAInd Platypus 2 model. Designed for chat and code generation."}, {title: "nexusraven", description: "Nexus Raven is a 13B instruction tuned model for function calling tasks. "}, {title: "goliath", description: "A language model created by combining two fine-tuned Llama 2 70B models into one."}, {title: "notux", description: "A top-performing mixture of experts model, fine-tuned with high-quality data."}, {title: "alfred", description: "A robust conversational model designed to be used for both chat and instruct use cases."}, {title: "megadolphin", description: "MegaDolphin-2.2-120b is a transformation of Dolphin-2.2-70b created by interleaving the model with itself."}, {title: "xwinlm", description: "Conversational model based on Llama 2 that performs competitively on various benchmarks."}, {title: "wizardlm", description: "General use 70 billion parameter model based on Llama 2."}, {title: "notus", description: "A 7B chat model fine-tuned with high-quality data and based on Zephyr."}, {title: "duckdb-nsql", description: "7B parameter text-to-SQL model made by MotherDuck and Numbers Station."}, {title: "all-minilm", description: "Embedding models on very large sentence level datasets."}, {title: "dolphincoder", description: "An uncensored variant of the Dolphin model family that excels at coding, based on StarCoder2."}}
	tokens = []string{"**", "Generic", " type", " constraints", "**", " allow", " you", " to", " specify", " constraints", " on", " a", " type", " that", " can", " vary", " depending", " on", " the", " specific", " type", " being", " instantiated", ".", "\n\n", "**", "Syntax", ":**", "\n\n", "```", "go", "\n", "type", " Name", "[", "T", " any", "]", " string", "\n", "```", "\n\n", "**", "Parameters", ":**", "\n\n", "*", " `", "T", "`:", " The", " type", " variable", ".", " It", " can", " be", " any", " type", ",", " including", " primitive", " types", ",", " structures", ",", " and", " functions", ".", "\n\n", "**", "Examples", ":**", "\n\n", "*", " ", "Integer", " constraint", ":**", "\n", "```", "go", "\n", "type", " Age", "[", "T", " int", "]", " int", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " an", " integer", " type", ".", "\n\n", "*", " ", "String", " constraint", ":**", "\n", "```", "go", "\n", "type", " Name", "[", "T", " string", "]", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " a", " string", " type", ".", "\n\n", "*", " ", "Struct", " constraint", ":**", "\n", "```", "go", "\n", "type", " User", "[", "T", " struct", "]", " {", "\n", "  ", "Name", " string", "\n", "  ", "Age", "  ", "int", "\n", "}", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " a", " struct", " type", " with", " at", " least", " two", " fields", " named", " `", "Name", "`", " and", " `", "Age", "`.", "\n\n", "*", " ", "Function", " constraint", ":**", "\n", "```", "go", "\n", "type", " Calculator", "[", "T", " any", "]", " func", "(", "T", ",", " T", ")", " T", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " a", " type", " that", " implements", " the", " `", "Calculator", "`", " interface", ".", "\n\n", "**", "Benefits", " of", " using", " generic", " type", " constraints", ":**", "\n\n", "*", " ", "Code", " reus", "ability", ":**", " You", " can", " apply", " the", " same", " constraint", " to", " multiple", " types", ",", " reducing", " code", " duplication", ".", "\n", "*", " ", "Type", " safety", ":**", " Constraints", " ensure", " that", " only", " valid", " types", " are", " used", ",", " preventing", " runtime", " errors", ".", "\n", "*", " ", "Improved", " maintain", "ability", ":**", " By", " separating", " the", " constraint", " from", " the", " type", ",", " it", " becomes", " easier", " to", " understand", " and", " modify", ".", "\n\n", "**", "Note", ":**", "\n\n", "*", " Generic", " type", " constraints", " are", " not", " applicable", " to", " primitive", " types", " (", "e", ".", "g", ".,", " `", "int", "`,", " `", "string", "`).", "\n", "*", " Constraints", " can", " be", " applied", " to", " function", " types", " only", " if", " the", " function", " is", " generic", ".", "\n", "*", " Constraints", " can", " be", " used", " with", " type", " parameters", ",", " allowing", " you", " to", " specify", " different", " constraints", " for", " different", " types", "."}
)
