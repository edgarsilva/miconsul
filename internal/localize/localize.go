// Code generated by go-localize; DO NOT EDIT.
// This file was generated by robots at
// 2024-05-16 20:36:37.132737272 -0600 CST m=+0.000630891

package localize

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

var localizations = map[string]string{
	"en-US.btn.back":               "Back",
	"en-US.btn.cancel":             "Cancel",
	"en-US.btn.confirm":            "Confirm",
	"en-US.btn.login":              "Login",
	"en-US.btn.logout":             "Logout",
	"en-US.btn.new_patient":        "Add new",
	"en-US.btn.reschedule":         "Reschedule",
	"en-US.btn.save":               "Save",
	"en-US.btn.signup":             "Signup",
	"en-US.btn.update":             "Update",
	"en-US.btn.view_all":           "View all",
	"en-US.btn.view_appointments":  "View appointments",
	"en-US.nav.appointments":       "Appointments",
	"en-US.nav.clinics":            "Clinics",
	"en-US.nav.exp_rev":            "Expents/Revenue",
	"en-US.nav.next_5":             "Next 5",
	"en-US.nav.patients":           "Patients",
	"en-US.nav.this_month":         "This month",
	"en-US.nav.this_week":          "This week",
	"en-US.nav.today":              "Today",
	"en-US.str.age":                "Age",
	"en-US.str.all_appointments":   "All appointments",
	"en-US.str.all_revenue":        "All revenue",
	"en-US.str.appointment_today":  "Appt. Today",
	"en-US.str.appointments":       "Appointments",
	"en-US.str.back":               "Back",
	"en-US.str.canceled":           "canceled",
	"en-US.str.change":             "Change",
	"en-US.str.clinics":            "Clinics",
	"en-US.str.confirmed":          "confirmed",
	"en-US.str.cover_photo":        "Cover photo",
	"en-US.str.create_new_patient": "Add a new patient",
	"en-US.str.date":               "Date",
	"en-US.str.decreased_by":       "Decreased by",
	"en-US.str.email_address":      "Email address",
	"en-US.str.exp_rev":            "Expents/Revenue",
	"en-US.str.first_name":         "First name",
	"en-US.str.increased_by":       "Increased by",
	"en-US.str.last_appointment":   "Last appointment ",
	"en-US.str.last_name":          "Last name",
	"en-US.str.location":           "Location",
	"en-US.str.login":              "Login",
	"en-US.str.logout":             "Logout",
	"en-US.str.ocupation":          "Ocupation",
	"en-US.str.or_drag_and_drop":   "or drag and drop",
	"en-US.str.patients":           "Patients",
	"en-US.str.pending":            "pending",
	"en-US.str.personal_info":      "Personal information",
	"en-US.str.personal_info_desc": "Use a permanent email address where you can receive mail.",
	"en-US.str.profile_pic":        "Profile Pic",
	"en-US.str.rescheduled":        "rescheduled",
	"en-US.str.revenue_month":      "Revenue this month",
	"en-US.str.save":               "Save",
	"en-US.str.signup":             "Signup",
	"en-US.str.stats":              "Statistics",
	"en-US.str.up_to_10mb":         "up to 10MB",
	"en-US.str.upload_file":        "Upload a file",
	"es-MX.btn.back":               "Regresar",
	"es-MX.btn.cancel":             "Cancelar",
	"es-MX.btn.confirm":            "Confirmar",
	"es-MX.btn.login":              "Iniciar sesion",
	"es-MX.btn.logout":             "Salir",
	"es-MX.btn.new_patient":        "Agregar nuevo",
	"es-MX.btn.reschedule":         "Reagendar",
	"es-MX.btn.save":               "Guardar",
	"es-MX.btn.signup":             "Registrate",
	"es-MX.btn.update":             "Guardar",
	"es-MX.btn.view_all":           "Ver todo",
	"es-MX.btn.view_appointments":  "Ver citas",
	"es-MX.nav.appointments":       "Citas",
	"es-MX.nav.clinics":            "Consultorios",
	"es-MX.nav.exp_rev":            "Contabilidad",
	"es-MX.nav.next_5":             "Siguientes 5",
	"es-MX.nav.patients":           "Pacientes",
	"es-MX.nav.this_month":         "Mi mes",
	"es-MX.nav.this_week":          "Mi semana",
	"es-MX.nav.today":              "Mi dia",
	"es-MX.str.age":                "Edad",
	"es-MX.str.all_appointments":   "Todas las citas",
	"es-MX.str.all_revenue":        "Contabilidad",
	"es-MX.str.appointment_today":  "Cita Hoy",
	"es-MX.str.appointments":       "Citas",
	"es-MX.str.back":               "Regresar",
	"es-MX.str.canceled":           "cancelada",
	"es-MX.str.change":             "Cambiar",
	"es-MX.str.clinics":            "Consultorios",
	"es-MX.str.confirmed":          "confirmada",
	"es-MX.str.cover_photo":        "Archivos adjuntos",
	"es-MX.str.create_new_patient": "Agregar un nuevo paciente",
	"es-MX.str.date":               "fecha",
	"es-MX.str.decreased_by":       "Decremento de",
	"es-MX.str.email_address":      "Correo electronico",
	"es-MX.str.exp_rev":            "Gastos/Ganancias",
	"es-MX.str.first_name":         "Nombres",
	"es-MX.str.increased_by":       "Incremento de",
	"es-MX.str.last_appointment":   "Ultima cita",
	"es-MX.str.last_name":          "Apellidos",
	"es-MX.str.location":           "Ubicacion",
	"es-MX.str.login":              "Iniciar sesion",
	"es-MX.str.logout":             "Cerrar sesion",
	"es-MX.str.ocupation":          "Ocupacion",
	"es-MX.str.or_drag_and_drop":   "o arrastrala aqui",
	"es-MX.str.patients":           "Pacientes",
	"es-MX.str.pending":            "pendiente",
	"es-MX.str.personal_info":      "Informacion personal",
	"es-MX.str.personal_info_desc": "Usa una direccion de correo en la que puedas recibir notificaciones.",
	"es-MX.str.profile_pic":        "Foto de perfil",
	"es-MX.str.rescheduled":        "reagendada",
	"es-MX.str.revenue_month":      "Ganancias del mes",
	"es-MX.str.save":               "Guardar",
	"es-MX.str.signup":             "Registrate",
	"es-MX.str.stats":              "Datos mensuales",
	"es-MX.str.up_to_10mb":         "hasta 10MB",
	"es-MX.str.upload_file":        "Sube un archivo",
}

type Replacements map[string]interface{}

type Localizer struct {
	Locale         string
	FallbackLocale string
	Localizations  map[string]string
}

func New(locale string, fallbackLocale string) *Localizer {
	t := &Localizer{Locale: locale, FallbackLocale: fallbackLocale}
	t.Localizations = localizations
	return t
}

func (t Localizer) SetLocales(locale, fallback string) Localizer {
	t.Locale = locale
	t.FallbackLocale = fallback
	return t
}

func (t Localizer) SetLocale(locale string) Localizer {
	t.Locale = locale
	return t
}

func (t Localizer) SetFallbackLocale(fallback string) Localizer {
	t.FallbackLocale = fallback
	return t
}

func (t Localizer) GetWithLocale(locale, key string, replacements ...*Replacements) string {
	str, ok := t.Localizations[t.getLocalizationKey(locale, key)]
	if !ok {
		str, ok = t.Localizations[t.getLocalizationKey(t.FallbackLocale, key)]
		if !ok {
			return key
		}
	}

	// If the str doesn't have any substitutions, no need to
	// template.Execute.
	if strings.Index(str, "}}") == -1 {
		return str
	}

	return t.replace(str, replacements...)
}

func (t Localizer) Get(key string, replacements ...*Replacements) string {
	str := t.GetWithLocale(t.Locale, key, replacements...)
	return str
}

func (t Localizer) getLocalizationKey(locale string, key string) string {
	return fmt.Sprintf("%v.%v", locale, key)
}

func (t Localizer) replace(str string, replacements ...*Replacements) string {
	b := &bytes.Buffer{}
	tmpl, err := template.New("").Parse(str)
	if err != nil {
		return str
	}

	replacementsMerge := Replacements{}
	for _, replacement := range replacements {
		for k, v := range *replacement {
			replacementsMerge[k] = v
		}
	}

	err = template.Must(tmpl, err).Execute(b, replacementsMerge)
	if err != nil {
		return str
	}
	buff := b.String()
	return buff
}
