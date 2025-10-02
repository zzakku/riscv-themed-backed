package ds

type CommandProgram struct {
	ID uint `gorm:"primaryKey"`
	// здесь создаем Unique key, указывая общий uniqueIndex
	CommandID uint `gorm:"not null;uniqueIndex:idx_command_program"`
	ProgramID uint `gorm:"not null;uniqueIndex:idx_command_program"`

	Operand int

	Command Command `gorm:"foreignKey:CommandID"`
	Program Program `gorm:"foreignKey:ProgramID"`
}
