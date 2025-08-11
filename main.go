//EASY

// defer close for everything that does db.Query
//Check for graceful error handling everywhere
//Create some basic snippets -- maybe a bandit walkthrough
// VarSave output directory

// MEDIUM

//Add unit tests
// add vscode style snippets
// consolidating duplicative code
// when editing the variable description field, all variable descriptions get changed. -- is this a bug in charm?
// rename project to ask

// HARD
// ultisnips -- do I care about the order of the snippets?  I think I probably do. -- might have to re-think the struct
// clipboard is not working -- not sure why.  Maybe would work better on raw linux?

//Roadmap
//Documentation -- descriptions for variables and snippets are limited to less than 255 characters.  This is not due to any technical limitation, but longer descriptions make the resulting metadata tables less appealing.  Also, if you cannot describe what the snippet does in less than 255 characters, you are probably being too ambitious.


package main

import (
	"go-ask/cmd"
)

func main() {
	cmd.Execute()
}
