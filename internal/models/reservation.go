package models

// ReservationInput deliberately excludes axle data, accessories and notes.
type ReservationInput struct {
	Area     string `json:"area"`
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Plate    string `json:"plate"`
	Brand    string `json:"brand"`
	Model    string `json:"model"`
	TireType string `json:"tire_type"`
	RimType  string `json:"rim_type"`
}

func (r *ReservationInput) Validate() map[string]string {
	s := StorageInput{Area: r.Area, Number: r.Number, Name: r.Name, Plate: r.Plate, Brand: r.Brand, Model: r.Model, TireType: r.TireType, RimType: r.RimType}
	fields := s.validate(false)
	r.Name = s.Name
	r.Plate = s.Plate
	r.Model = s.Model
	return fields
}
func (r ReservationInput) Place() string {
	return (StorageInput{Area: r.Area, Number: r.Number}).Place()
}
