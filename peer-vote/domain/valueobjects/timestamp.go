package valueobjects

import (
	"time"
)

// Timestamp representa um momento no tempo
type Timestamp struct {
	value time.Time
}

// NewTimestamp cria um novo timestamp
func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{value: t.UTC()}
}

// Now retorna o timestamp atual
func Now() Timestamp {
	return Timestamp{value: time.Now().UTC()}
}

// Unix cria um timestamp a partir de um valor Unix
func Unix(sec int64, nsec int64) Timestamp {
	return Timestamp{value: time.Unix(sec, nsec).UTC()}
}

// Time retorna o valor time.Time
func (t Timestamp) Time() time.Time {
	return t.value
}

// Unix retorna o timestamp Unix
func (t Timestamp) Unix() int64 {
	return t.value.Unix()
}

// UnixNano retorna o timestamp Unix em nanosegundos
func (t Timestamp) UnixNano() int64 {
	return t.value.UnixNano()
}

// String retorna a representação string do timestamp
func (t Timestamp) String() string {
	return t.value.Format(time.RFC3339)
}

// IsZero verifica se o timestamp é zero
func (t Timestamp) IsZero() bool {
	return t.value.IsZero()
}

// Before verifica se este timestamp é anterior ao outro
func (t Timestamp) Before(other Timestamp) bool {
	return t.value.Before(other.value)
}

// After verifica se este timestamp é posterior ao outro
func (t Timestamp) After(other Timestamp) bool {
	return t.value.After(other.value)
}

// Equal verifica se dois timestamps são iguais
func (t Timestamp) Equal(other Timestamp) bool {
	return t.value.Equal(other.value)
}

// Add adiciona uma duração ao timestamp
func (t Timestamp) Add(d time.Duration) Timestamp {
	return Timestamp{value: t.value.Add(d)}
}

// Sub subtrai outro timestamp deste
func (t Timestamp) Sub(other Timestamp) time.Duration {
	return t.value.Sub(other.value)
}

// IsValid verifica se o timestamp é válido
func (t Timestamp) IsValid() bool {
	// Um timestamp é válido se não for zero e não for muito no futuro
	if t.IsZero() {
		return false
	}
	
	// Não deve ser mais de 1 hora no futuro
	maxFuture := time.Now().UTC().Add(time.Hour)
	return t.value.Before(maxFuture)
}

// Format formata o timestamp usando o layout especificado
func (t Timestamp) Format(layout string) string {
	return t.value.Format(layout)
}
