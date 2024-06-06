// Code generated by go-localize; DO NOT EDIT.
// This file was generated by robots at
// 2024-06-06 10:33:14.865961956 -0600 CST m=+0.001097570

package localize

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

var localizations = map[string]string{
	"en-US.btn.appointments":                             "Appointments",
	"en-US.btn.back":                                     "Back",
	"en-US.btn.cancel":                                   "Cancel",
	"en-US.btn.change":                                   "Change",
	"en-US.btn.choose_file":                              "Choose file",
	"en-US.btn.confirm":                                  "Confirm",
	"en-US.btn.done":                                     "Done",
	"en-US.btn.edit":                                     "Edit",
	"en-US.btn.login":                                    "Login",
	"en-US.btn.logout":                                   "Logout",
	"en-US.btn.new":                                      "Add new",
	"en-US.btn.new_patient":                              "Add new",
	"en-US.btn.open":                                     "Open",
	"en-US.btn.remove":                                   "Remove",
	"en-US.btn.reschedule":                               "Reschedule",
	"en-US.btn.save":                                     "Save",
	"en-US.btn.signup":                                   "Signup",
	"en-US.btn.update":                                   "Update",
	"en-US.btn.view_all":                                 "View all",
	"en-US.btn.view_appointments":                        "View appointments",
	"en-US.email.appointment_today_body":                 "You have an appointment today, here are the details:",
	"en-US.email.cancel_appointment":                     "Cancel my appointment",
	"en-US.email.confirm_appointment":                    "Confirm my assistance",
	"en-US.email.confirm_appointment_title":              "You got an appointment today!",
	"en-US.email.greeting":                               "Hello",
	"en-US.email.i_want_to":                              "I want to:",
	"en-US.email.reschudule_appointment":                 "Reschudule my appointment",
	"en-US.nav.appointments":                             "Appointments",
	"en-US.nav.clinics":                                  "Clinics",
	"en-US.nav.exp_rev":                                  "Expents/Revenue",
	"en-US.nav.my_day":                                   "My day",
	"en-US.nav.my_month":                                 "My month",
	"en-US.nav.my_week":                                  "My week",
	"en-US.nav.next_5":                                   "Next 5",
	"en-US.nav.patients":                                 "Patients",
	"en-US.str.address":                                  "Address",
	"en-US.str.address_city":                             "City",
	"en-US.str.address_country":                          "Country",
	"en-US.str.address_desc":                             "Street name and number, city, state or province, country (optional) and zip code.",
	"en-US.str.address_state_province":                   "State / Province",
	"en-US.str.address_street":                           "Street address",
	"en-US.str.address_zip_code":                         "Zip / Postal Code",
	"en-US.str.age":                                      "Age",
	"en-US.str.all_appointments":                         "All appointments",
	"en-US.str.all_revenue":                              "All revenue",
	"en-US.str.apnt_step_1":                              "Step 1",
	"en-US.str.apnt_step_1_desc":                         "Select the clinic",
	"en-US.str.apnt_step_2":                              "Step 2",
	"en-US.str.apnt_step_2_desc":                         "Select the patient",
	"en-US.str.apnt_step_3":                              "Step 3",
	"en-US.str.apnt_step_3_desc":                         "Fill in the appointment details",
	"en-US.str.appointment_confirmed":                    "Your appointment has been confirmed.",
	"en-US.str.appointment_send_email":                   "Send email",
	"en-US.str.appointment_send_email_desc":              "Send an email to let the user know a new appointment has been schedulet (includes links where they can Confirm/Cancel/Reschedule)",
	"en-US.str.appointment_today":                        "Appt. Today",
	"en-US.str.appointments":                             "Appointments",
	"en-US.str.are_you_sure":                             "Are you sure?",
	"en-US.str.back":                                     "Back",
	"en-US.str.background":                               "Background",
	"en-US.str.background_ph":                            "Medical history, previous deceases and episodes, triggers, family status, etc...",
	"en-US.str.booked_at":                                "Booked at",
	"en-US.str.cancel_appointment_confirmation":          "Are you sure you want to cancel your confirmation?",
	"en-US.str.canceled":                                 "canceled",
	"en-US.str.change":                                   "Change",
	"en-US.str.clinic_info":                              "Clinic info",
	"en-US.str.clinic_info_desc":                         "Informacion basica del consultorio.",
	"en-US.str.clinic_name":                              "Name",
	"en-US.str.clinic_pic":                               "Thumbnail pic",
	"en-US.str.clinics":                                  "Clinics",
	"en-US.str.conclusions":                              "Conclusions",
	"en-US.str.confirmed":                                "confirmed",
	"en-US.str.cost":                                     "Cost",
	"en-US.str.cover_photo":                              "Cover photo",
	"en-US.str.create_new_appointment":                   "New",
	"en-US.str.create_new_patient":                       "New",
	"en-US.str.current_session":                          "Current session",
	"en-US.str.current_session_desc":                     "Fill in the appointment details.",
	"en-US.str.date":                                     "Date",
	"en-US.str.decreased_by":                             "Decreased by",
	"en-US.str.done":                                     "done",
	"en-US.str.duration":                                 "Duration",
	"en-US.str.edit_appointment":                         "Edit appointment",
	"en-US.str.edit_clinic":                              "Edit clinic",
	"en-US.str.edit_patient":                             "Edit patient",
	"en-US.str.email":                                    "Email",
	"en-US.str.email_address":                            "Email address",
	"en-US.str.exp_rev":                                  "Expents/Revenue",
	"en-US.str.facebook":                                 "Facebook",
	"en-US.str.family_history":                           "Family history",
	"en-US.str.family_history_ph":                        "Relevant information about family background",
	"en-US.str.favorite":                                 "Favorite",
	"en-US.str.first_name":                               "First name",
	"en-US.str.increased_by":                             "Increased by",
	"en-US.str.instagram":                                "Instagram",
	"en-US.str.last_appointment":                         "Last appointment ",
	"en-US.str.last_name":                                "Last name",
	"en-US.str.last_session":                             "Last session",
	"en-US.str.last_session_desc":                        "Summary and notes about the last session.",
	"en-US.str.location":                                 "Location",
	"en-US.str.login":                                    "Login",
	"en-US.str.logout":                                   "Logout",
	"en-US.str.medical_profile":                          "Medical profile",
	"en-US.str.medical_profile_desc":                     "This information is private, it will only be shared with the appropiate personel.",
	"en-US.str.new_appointment":                          "New appointment",
	"en-US.str.new_clinic":                               "New clinic",
	"en-US.str.new_patient":                              "New patient",
	"en-US.str.no_prev_apnt":                             "No previous appointments.",
	"en-US.str.notes":                                    "Notes",
	"en-US.str.notes_ph":                                 "Other information considered relevant or related",
	"en-US.str.nothing_found":                            "Nothing found with that query",
	"en-US.str.notifications":                            "Notifications",
	"en-US.str.notifications_about":                      "Notify about",
	"en-US.str.notifications_appointment_lifecycle":      "Create/Cancel/Reschedule appointments",
	"en-US.str.notifications_appointment_lifecycle_desc": "Notify when a new appointment is created, canceled or rescheduled.",
	"en-US.str.notifications_desc":                       "We'll let the patient know about important changes to their appointments, but you can pick what they are notified about.",
	"en-US.str.notifications_via":                        "Send notifications via",
	"en-US.str.observations":                             "Observations",
	"en-US.str.ocupation":                                "Ocupation",
	"en-US.str.or_drag_and_drop":                         "or drag and drop",
	"en-US.str.patient":                                  "Patient",
	"en-US.str.patient_info":                             "Patient",
	"en-US.str.patient_info_desc":                        "What patient is this appointment for.",
	"en-US.str.patients":                                 "Patients",
	"en-US.str.pending":                                  "pending",
	"en-US.str.personal_info":                            "Personal information",
	"en-US.str.personal_info_desc":                       "Use a permanent email address where you can receive mail.",
	"en-US.str.phone":                                    "Phone",
	"en-US.str.profile_pic":                              "Profile Pic",
	"en-US.str.rescheduled":                              "rescheduled",
	"en-US.str.revenue_month":                            "Revenue this month",
	"en-US.str.save":                                     "Save",
	"en-US.str.search_clinics":                           "Begin typing to search clinics...",
	"en-US.str.search_patients":                          "Begin typing to search patients...",
	"en-US.str.session_info":                             "Session",
	"en-US.str.session_info_desc":                        "Take observations and notes, fill in the session details.",
	"en-US.str.show_completed":                           "Show Completed",
	"en-US.str.show_done":                                "Show done",
	"en-US.str.signup":                                   "Signup",
	"en-US.str.stats":                                    "Statistics",
	"en-US.str.status":                                   "Status",
	"en-US.str.summary":                                  "Summary",
	"en-US.str.summary_ph":                               "A brief description of the session for future reference, preferably 2 or 3 lines long, it will be referenced in the next session.",
	"en-US.str.telegram":                                 "Telegram",
	"en-US.str.time_and_date":                            "Time and date",
	"en-US.str.up_to_10mb":                               "up to 10MB",
	"en-US.str.upload_file":                              "Upload a file",
	"en-US.str.view_more":                                "View more",
	"en-US.str.viewed":                                   "viewed",
	"en-US.str.whatsapp":                                 "Whatsapp",
	"en-US.str.yes_cancel_my_appointment":                "Yes cancel my appointment",
	"es-MX.btn.appointments":                             "Citas",
	"es-MX.btn.back":                                     "Regresar",
	"es-MX.btn.cancel":                                   "Cancelar",
	"es-MX.btn.change":                                   "Cambiar",
	"es-MX.btn.choose_file":                              "Elegir archivo",
	"es-MX.btn.confirm":                                  "Confirmar",
	"es-MX.btn.done":                                     "Concluir",
	"es-MX.btn.edit":                                     "Editar",
	"es-MX.btn.login":                                    "Iniciar sesion",
	"es-MX.btn.logout":                                   "Salir",
	"es-MX.btn.new":                                      "Nuevo",
	"es-MX.btn.new_patient":                              "Nuevo",
	"es-MX.btn.open":                                     "Abrir",
	"es-MX.btn.remove":                                   "Quitar",
	"es-MX.btn.reschedule":                               "Reagendar",
	"es-MX.btn.save":                                     "Guardar",
	"es-MX.btn.signup":                                   "Registrate",
	"es-MX.btn.update":                                   "Actualizar",
	"es-MX.btn.view_all":                                 "Ver todo",
	"es-MX.btn.view_appointments":                        "Ver citas",
	"es-MX.email.appointment_today_body":                 "Tienes una cita hoy, estos son los detalles de tu cita:",
	"es-MX.email.cancel_appointment":                     "Cancelar mi cita",
	"es-MX.email.confirm_appointment":                    "Confirmar mi asistencia!",
	"es-MX.email.confirm_appointment_title":              "Tienes una cita hoy!",
	"es-MX.email.greeting":                               "Hola",
	"es-MX.email.i_want_to":                              "Me gustaria:",
	"es-MX.email.reschudule_appointment":                 "Reagendar mi cita",
	"es-MX.nav.appointments":                             "Citas",
	"es-MX.nav.clinics":                                  "Consultorios",
	"es-MX.nav.exp_rev":                                  "Contabilidad",
	"es-MX.nav.my_day":                                   "Mi dia",
	"es-MX.nav.my_month":                                 "Mi mes",
	"es-MX.nav.my_week":                                  "Mi semana",
	"es-MX.nav.next_5":                                   "Siguientes 5",
	"es-MX.nav.patients":                                 "Pacientes",
	"es-MX.str.address":                                  "Direccion",
	"es-MX.str.address_city":                             "Ciudad",
	"es-MX.str.address_country":                          "Pais",
	"es-MX.str.address_desc":                             "Calle, numero exterior, ciudad, estado, pais (opcional) y codigo postal.",
	"es-MX.str.address_state":                            "Estado",
	"es-MX.str.address_street":                           "Calle y Numero",
	"es-MX.str.address_zip_code":                         "Codigo postal",
	"es-MX.str.age":                                      "Edad",
	"es-MX.str.all_appointments":                         "Todas las citas",
	"es-MX.str.all_revenue":                              "Contabilidad",
	"es-MX.str.apnt_step_1":                              "Paso 1",
	"es-MX.str.apnt_step_1_desc":                         "Selecciona el consultorio",
	"es-MX.str.apnt_step_2":                              "Paso 2",
	"es-MX.str.apnt_step_2_desc":                         "Selecciona el paciente",
	"es-MX.str.apnt_step_3":                              "Paso 3",
	"es-MX.str.apnt_step_3_desc":                         "Llena los detalles de tu cita",
	"es-MX.str.appointment_confirmed":                    "Tu cita ha sido confirmada.",
	"es-MX.str.appointment_send_email":                   "Enviar correo electronico",
	"es-MX.str.appointment_send_email_desc":              "Enviar un correo al paciente avisando que se ha agendado una nueva cita (incluye links para confimar, cancelar o reagendar)",
	"es-MX.str.appointment_today":                        "Cita Hoy",
	"es-MX.str.appointments":                             "Citas",
	"es-MX.str.are_you_sure":                             "Estas seguro?",
	"es-MX.str.back":                                     "Regresar",
	"es-MX.str.background":                               "Antedecentes",
	"es-MX.str.background_ph":                            "Historial medico, enfermedades previas, episodios pasados, detonantes, situacion familiar, etc...",
	"es-MX.str.booked_at":                                "Agendar para el",
	"es-MX.str.cancel_appointment_confirmation":          "Estas seguro de querer cancelar tu cita?",
	"es-MX.str.canceled":                                 "cancelada",
	"es-MX.str.change":                                   "Cambiar",
	"es-MX.str.clinic_info":                              "Datos del consultorio",
	"es-MX.str.clinic_info_desc":                         "Informacion basica del consultorio.",
	"es-MX.str.clinic_name":                              "Nombre",
	"es-MX.str.clinic_pic":                               "Foto de perfil",
	"es-MX.str.clinics":                                  "Consultorios",
	"es-MX.str.conclusions":                              "Conclusiones",
	"es-MX.str.conclusions_ph":                           "",
	"es-MX.str.confirmed":                                "confirmada",
	"es-MX.str.cost":                                     "Costo",
	"es-MX.str.cover_photo":                              "Archivos adjuntos",
	"es-MX.str.create_new_appointment":                   "Agregar cita nueva",
	"es-MX.str.create_new_patient":                       "Agregar un paciente nuevo",
	"es-MX.str.current_session":                          "Detalles de la sesion",
	"es-MX.str.current_session_desc":                     "Llena los detalles de tu cita.",
	"es-MX.str.date":                                     "fecha",
	"es-MX.str.decreased_by":                             "Decremento de",
	"es-MX.str.done":                                     "concluida",
	"es-MX.str.duration":                                 "Duracion",
	"es-MX.str.edit_appointment":                         "Editar cita",
	"es-MX.str.edit_clinic":                              "Editar consulorio",
	"es-MX.str.edit_patient":                             "Editar paciente",
	"es-MX.str.email":                                    "Correo electronico",
	"es-MX.str.email_address":                            "Correo electronico",
	"es-MX.str.exp_rev":                                  "Gastos/Ganancias",
	"es-MX.str.facebook":                                 "Facebook",
	"es-MX.str.family_history":                           "Historial familiar",
	"es-MX.str.family_history_ph":                        "Informacion relevante sobre los antecedentes familiares",
	"es-MX.str.favorite":                                 "Favorito",
	"es-MX.str.first_name":                               "Nombres",
	"es-MX.str.increased_by":                             "Incremento de",
	"es-MX.str.instagram":                                "Instagram",
	"es-MX.str.last_appointment":                         "Ultima cita",
	"es-MX.str.last_name":                                "Apellidos",
	"es-MX.str.last_session":                             "Ultima sesion",
	"es-MX.str.last_session_desc":                        "Resumen y notas de la ultima sesion.",
	"es-MX.str.location":                                 "Ubicacion",
	"es-MX.str.login":                                    "Iniciar sesion",
	"es-MX.str.logout":                                   "Cerrar sesion",
	"es-MX.str.medical_profile":                          "Historial medico",
	"es-MX.str.medical_profile_desc":                     "Esta informacion es privada, se compartira solo con personal autorizado.",
	"es-MX.str.new_appointment":                          "Nueva cita",
	"es-MX.str.new_clinic":                               "Nuevo consultorio",
	"es-MX.str.new_patient":                              "Paciente nuevo",
	"es-MX.str.no_prev_apnt":                             "No hay citas previas.",
	"es-MX.str.notes":                                    "Notas",
	"es-MX.str.notes_ph":                                 "Informacion extra que podria ser relevante para esta cita, o la siguiente (seran incluidas en el resumen del paciente en la siguiente sesion).",
	"es-MX.str.nothing_found":                            "No hay resultados con esa busqueda",
	"es-MX.str.notifications":                            "Notificaciones",
	"es-MX.str.notifications_about":                      "Enviar notificaciones",
	"es-MX.str.notifications_appointment_lifecycle":      "Al crear, cancelar o reagendar citas",
	"es-MX.str.notifications_appointment_lifecycle_desc": "Notificar al paciente cuando creo, cancelo o reagendo citas.",
	"es-MX.str.notifications_desc":                       "Notificaremos al paciente sobre cambios importantes a su cita, tambien puedes escoger a traves de que plataforma sera notificado",
	"es-MX.str.notifications_via":                        "Enviar notificaciones por:",
	"es-MX.str.observations":                             "Observaciones",
	"es-MX.str.observations_ph":                          "",
	"es-MX.str.ocupation":                                "Ocupacion",
	"es-MX.str.or_drag_and_drop":                         "o arrastralo aqui",
	"es-MX.str.patient":                                  "Paciente",
	"es-MX.str.patient_info":                             "Paciente",
	"es-MX.str.patient_info_desc":                        "Selecciona el paciente para la cita.",
	"es-MX.str.patients":                                 "Pacientes",
	"es-MX.str.pending":                                  "pendiente",
	"es-MX.str.personal_info":                            "Informacion personal",
	"es-MX.str.personal_info_desc":                       "Usa una direccion de correo en la que puedas recibir notificaciones.",
	"es-MX.str.phone":                                    "Telefono",
	"es-MX.str.profile_pic":                              "Foto de perfil",
	"es-MX.str.rescheduled":                              "reagendada",
	"es-MX.str.revenue_month":                            "Ganancias del mes",
	"es-MX.str.save":                                     "Guardar",
	"es-MX.str.search_clinics":                           "Escribe para buscar consultorios...",
	"es-MX.str.search_patients":                          "Escribe para buscar pacientes...",
	"es-MX.str.session_info":                             "Sesion",
	"es-MX.str.session_info_desc":                        "Registra los detalles de la sesion.",
	"es-MX.str.show_completed":                           "Mostrar completados",
	"es-MX.str.show_done":                                "Mostrar terminados",
	"es-MX.str.signup":                                   "Registrate",
	"es-MX.str.social_media":                             "Redes sociales",
	"es-MX.str.social_media_desc":                        "Todas las redes sociales que quieres ligar a tu consultorio.",
	"es-MX.str.stats":                                    "Datos mensuales",
	"es-MX.str.status":                                   "Estado",
	"es-MX.str.summary":                                  "Resumen",
	"es-MX.str.summary_ph":                               "Una breve descripcion de la session, 2 o 3 lineas, sera incluida junto a los detalles del paciente en la siguiente sesion como referencia.",
	"es-MX.str.telegram":                                 "Telegram",
	"es-MX.str.time_and_date":                            "Fecha y Hora",
	"es-MX.str.up_to_10mb":                               "hasta 10MB",
	"es-MX.str.upload_file":                              "Sube un archivo",
	"es-MX.str.view_more":                                "Ver mas",
	"es-MX.str.viewed":                                   "vista",
	"es-MX.str.whatsapp":                                 "Whatsapp",
	"es-MX.str.yes_cancel_my_appointment":                "Si, deseo cancelar mi cita",
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
