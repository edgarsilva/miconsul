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
	Warning(class string) templ.Component
	Info(class string) templ.Component
	Eye(class string) templ.Component
	EyeOff(class string) templ.Component
	Key(class string) templ.Component
	ShieldCheck(class string) templ.Component
	Star(class string) templ.Component
	Image(class string) templ.Component
	Download(class string) templ.Component
	Upload(class string) templ.Component
	ChevronUp(class string) templ.Component
	ChevronLeft(class string) templ.Component
	ChevronRight(class string) templ.Component
	ArrowUp(class string) templ.Component
	ArrowDown(class string) templ.Component
	ArrowLeft(class string) templ.Component
	ArrowRight(class string) templ.Component
}

var (
	_ iconSet = lucideIcons{}
	_ iconSet = phosphorIcons{}
)

// icons is the active icon set. Change it to switch the whole app.
var icons iconSet = lucideIcons{}
