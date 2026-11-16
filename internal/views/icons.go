package views

import "github.com/a-h/templ"

// iconSet is one family of UI icons. Each method renders one icon with the
// given CSS classes. Every set must have every icon; the compiler checks it.
type iconSet interface {
	Email(class string) templ.Component
	Calendar(class string) templ.Component
	BookOpen(class string) templ.Component
	Signout(class string) templ.Component
	Signin(class string) templ.Component
	Search(class string) templ.Component
	MapPin(class string) templ.Component
	User(class string) templ.Component
	Stethoscope(class string) templ.Component
	UserAdd(class string) templ.Component
	Menu(class string) templ.Component
	Building(class string) templ.Component
	Phone(class string) templ.Component
	PlusCircle(class string) templ.Component
	Pencil(class string) templ.Component
	Close(class string) templ.Component
	CheckCircle(class string) templ.Component
	Trash(class string) templ.Component
	CloseCircle(class string) templ.Component
	ChevronDown(class string) templ.Component
}

var _ iconSet = heroIcons{}
var _ iconSet = lucideIcons{}

// icons is the active icon set. Change it to switch the whole app.
var icons iconSet = heroIcons{}
