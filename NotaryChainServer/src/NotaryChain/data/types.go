package data

type Block struct {
	BlockID			int64
	PreviousHash	[32]byte
	Entries			[]Entry
}

const (
	EmptyEntryType	= -1
	HashEntryType	=  0
	PlainEntryType	=  1
)

type Entry struct {
	EntryType		int8
}

type HashEntry struct {
	Entry
	Hash			[32]byte	// The hash data
}

type PlainEntry struct {
	Entry
	StructuredData	[]byte		// The data (could be hashes) to record
	Signatures      []Signature	// Optional signatures of the data
	TimeSamp        int64		// Unix Time
}

const (
	BadKeyType		= -1
	ECCKeyType		=  0
	RSAKeyType		=  1
)

type Key struct {
	KeyType			int8
	KeyData			[]byte
}

type Signature struct {
	PublicKey		Key
	SignedHash		[32]byte
}