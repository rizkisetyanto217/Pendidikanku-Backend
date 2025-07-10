package dto

import "masjidku_backend/internals/features/masjids/lecture_sessions/main/model"

type JSONBTeacher struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Converter
func (jt JSONBTeacher) ToModel() model.JSONBTeacher {
	return model.JSONBTeacher{
		ID:   jt.ID,
		Name: jt.Name,
	}
}

func FromModel(jt model.JSONBTeacher) JSONBTeacher {
	return JSONBTeacher{
		ID:   jt.ID,
		Name: jt.Name,
	}
}
