package main

import (
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

var (
	tokens = []string{"**", "Generic", " type", " constraints", "**", " allow", " you", " to", " specify", " constraints", " on", " a", " type", " that", " can", " vary", " depending", " on", " the", " specific", " type", " being", " instantiated", ".", "\n\n", "**", "Syntax", ":**", "\n\n", "```", "go", "\n", "type", " Name", "[", "T", " any", "]", " string", "\n", "```", "\n\n", "**", "Parameters", ":**", "\n\n", "*", " `", "T", "`:", " The", " type", " variable", ".", " It", " can", " be", " any", " type", ",", " including", " primitive", " types", ",", " structures", ",", " and", " functions", ".", "\n\n", "**", "Examples", ":**", "\n\n", "*", " ", "Integer", " constraint", ":**", "\n", "```", "go", "\n", "type", " Age", "[", "T", " int", "]", " int", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " an", " integer", " type", ".", "\n\n", "*", " ", "String", " constraint", ":**", "\n", "```", "go", "\n", "type", " Name", "[", "T", " string", "]", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " a", " string", " type", ".", "\n\n", "*", " ", "Struct", " constraint", ":**", "\n", "```", "go", "\n", "type", " User", "[", "T", " struct", "]", " {", "\n", "  ", "Name", " string", "\n", "  ", "Age", "  ", "int", "\n", "}", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " a", " struct", " type", " with", " at", " least", " two", " fields", " named", " `", "Name", "`", " and", " `", "Age", "`.", "\n\n", "*", " ", "Function", " constraint", ":**", "\n", "```", "go", "\n", "type", " Calculator", "[", "T", " any", "]", " func", "(", "T", ",", " T", ")", " T", "\n", "```", "\n\n", "This", " constraint", " ensures", " that", " `", "T", "`", " is", " a", " type", " that", " implements", " the", " `", "Calculator", "`", " interface", ".", "\n\n", "**", "Benefits", " of", " using", " generic", " type", " constraints", ":**", "\n\n", "*", " ", "Code", " reus", "ability", ":**", " You", " can", " apply", " the", " same", " constraint", " to", " multiple", " types", ",", " reducing", " code", " duplication", ".", "\n", "*", " ", "Type", " safety", ":**", " Constraints", " ensure", " that", " only", " valid", " types", " are", " used", ",", " preventing", " runtime", " errors", ".", "\n", "*", " ", "Improved", " maintain", "ability", ":**", " By", " separating", " the", " constraint", " from", " the", " type", ",", " it", " becomes", " easier", " to", " understand", " and", " modify", ".", "\n\n", "**", "Note", ":**", "\n\n", "*", " Generic", " type", " constraints", " are", " not", " applicable", " to", " primitive", " types", " (", "e", ".", "g", ".,", " `", "int", "`,", " `", "string", "`).", "\n", "*", " Constraints", " can", " be", " applied", " to", " function", " types", " only", " if", " the", " function", " is", " generic", ".", "\n", "*", " Constraints", " can", " be", " used", " with", " type", " parameters", ",", " allowing", " you", " to", " specify", " different", " constraints", " for", " different", " types", "."}
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
	gi.NewButton(leftFrame).SetText("Run module").Style(func(s *styles.Style) {
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
