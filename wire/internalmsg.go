package wire

// Commands used in bitcoin message headers which describe the type of message.
const (

	// Factom internal messages:
	CmdInt_FactoidObj   = "int_factoidobj"
	CmdInt_FactoidBlock = "int_fx_block"
	CmdInt_EOM          = "int_eom"
	CmdInt_DirBlock     = "int_dir_block"
)

// Block status code
const (
	BLOCK_QUERY_STATUS uint8 = iota
	BLOCK_BUILD_SUCCESS
	BLOCK_BUILD_FAILED
	BLOCK_NOT_FOUND
	BLOCK_NOT_VALID
)

// FtmInternalMsg is an interface that describes an internal factom message.
// The message is used for communication between two modules
type FtmInternalMsg interface {
	Command() string
}

// End-of-Minute internal message for time commnunications between Goroutines
type MsgInt_EOM struct {
	EOM_Type         byte
	NextDBlockHeight uint32
	//EC_Exchange_Rate uint64
}

// End-of-Minute internal message for time commnunications between Goroutines
func (msg *MsgInt_EOM) Command() string {
	return CmdInt_EOM
}

// Factoid block message for internal communication
type MsgInt_FactoidBlock struct {
	ShaHash            ShaHash
	BlockHeight        uint32
	FactoidBlockStatus byte
}

// Factoid block available: internal message for time commnunications between Goroutines
func (msg *MsgInt_FactoidBlock) Command() string {
	return CmdInt_FactoidBlock
}

// Dir block message for internal communication
type MsgInt_DirBlock struct {
	ShaHash *ShaHash
	//BlockHeight uint64
}

// Dir block available: internal message for time commnunications between Goroutines
func (msg *MsgInt_DirBlock) Command() string {
	return CmdInt_DirBlock
}
