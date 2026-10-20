// Package school is a sample whose join table name is changed at run time
// (db.SetupJoinTable), which only the config can tell the adapter.
package school

type Course struct {
	ID    uint
	Title string `gorm:"not null"`
}

type Student struct {
	ID      uint
	Courses []Course `gorm:"many2many:enrollments"`
}
