package sample

// Clean is a normal comment.
type Clean struct{}

/* This has **bold** text. */ // want "avoid Markdown formatting in Go comments"
type Bold struct{}

/* See [a link](http://example.com) here. */ // want "avoid Markdown formatting in Go comments"
type Link struct{}

// # Single heading is fine.
type SingleHeading struct{}
