package schemas

// Settings represents user settings.
// swagger:model
type Settings struct {
	// Theme of the user interface. 0 = Light, 1 = Dark.
	// required: true
	// nullable: false
	// minimum: 0
	// maximum: 1
	Theme int `json:"theme"`
}

// UpdateSettingsInput
// swagger:parameters UpdateSettings
type UpdateSettingsInput struct {
	// Settings data to update
	// required: true
	// nullable: false
	// in: body
	Body Settings
}
