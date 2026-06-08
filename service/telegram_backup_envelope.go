package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"

	"github.com/deposist/s-ui-x/util/common"
	"golang.org/x/crypto/argon2"
)

const (
	TelegramBackupMagic           = "SUI-TGBKP\x00"
	TelegramBackupEnvelopeVersion = byte(1)
	TelegramBackupKDFArgon2ID     = byte(1)

	telegramBackupMagicSize     = 10
	telegramBackupVersionSize   = 1
	telegramBackupKDFIDSize     = 1
	telegramBackupKDFParamsSize = 16
	telegramBackupSaltSize      = 16
	telegramBackupNonceSize     = 12
	telegramBackupHeaderSize    = telegramBackupMagicSize +
		telegramBackupVersionSize +
		telegramBackupKDFIDSize +
		telegramBackupKDFParamsSize +
		telegramBackupSaltSize +
		telegramBackupNonceSize

	telegramBackupArgon2MemoryKiB   = 64 * 1024
	telegramBackupArgon2Iterations  = 3
	telegramBackupArgon2Parallelism = 1
	telegramBackupKeySize           = 32
)

var (
	ErrTelegramBackupDecryptionFailed = errors.New("decryption_failed")
	ErrTelegramBackupInvalidEnvelope  = errors.New("invalid_backup_envelope")
)

type TelegramBackupKDFParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
}

type TelegramBackupEnvelopeHeader struct {
	Version      byte
	KDFID        byte
	KDFParams    TelegramBackupKDFParams
	RawKDFParams [telegramBackupKDFParamsSize]byte
	Salt         [telegramBackupSaltSize]byte
	Nonce        [telegramBackupNonceSize]byte
}

var telegramBackupDefaultKDFParams = TelegramBackupKDFParams{
	MemoryKiB:   telegramBackupArgon2MemoryKiB,
	Iterations:  telegramBackupArgon2Iterations,
	Parallelism: telegramBackupArgon2Parallelism,
}

// Backup envelope layout, fixed for version 1:
//
//	magic          10 bytes  "SUI-TGBKP\x00"
//	version         1 byte   currently 0x01
//	kdf-id          1 byte   0x01 = Argon2id
//	kdf-params     16 bytes  for Argon2id:
//	                         uint32 memoryKiB, uint32 iterations,
//	                         uint8 parallelism, 7 reserved zero bytes
//	salt           16 bytes  random per envelope
//	nonce          12 bytes  AES-GCM nonce, random per envelope
//	ciphertext+tag  N bytes  AES-256-GCM output
//
// AES-GCM authenticates the whole header through nonce as AAD.
func BuildTelegramBackupEnvelope(plaintext []byte, passphrase []byte) ([]byte, error) {
	return buildTelegramBackupEnvelope(plaintext, passphrase, rand.Reader)
}

func OpenTelegramBackupEnvelope(envelope []byte, passphrase []byte) ([]byte, error) {
	header, err := ParseTelegramBackupEnvelopeHeader(envelope)
	if err != nil {
		return nil, err
	}
	if header.Version != TelegramBackupEnvelopeVersion || header.KDFID != TelegramBackupKDFArgon2ID {
		return nil, ErrTelegramBackupInvalidEnvelope
	}
	params := header.KDFParams
	if err := validateTelegramBackupKDFParams(params); err != nil {
		return nil, err
	}
	key := deriveTelegramBackupKey(passphrase, header.Salt[:], params)
	defer zeroBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrTelegramBackupDecryptionFailed
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, telegramBackupNonceSize)
	if err != nil {
		return nil, ErrTelegramBackupDecryptionFailed
	}
	headerBytes := envelope[:telegramBackupHeaderSize]
	ciphertext := envelope[telegramBackupHeaderSize:]
	plaintext, err := gcm.Open(nil, header.Nonce[:], ciphertext, headerBytes)
	if err != nil {
		return nil, ErrTelegramBackupDecryptionFailed
	}
	return plaintext, nil
}

func ParseTelegramBackupEnvelopeHeader(envelope []byte) (TelegramBackupEnvelopeHeader, error) {
	var header TelegramBackupEnvelopeHeader
	if len(envelope) < telegramBackupHeaderSize {
		return header, ErrTelegramBackupInvalidEnvelope
	}
	if !IsTelegramBackupEnvelope(envelope) {
		return header, ErrTelegramBackupInvalidEnvelope
	}
	offset := telegramBackupMagicSize
	header.Version = envelope[offset]
	offset += telegramBackupVersionSize
	header.KDFID = envelope[offset]
	offset += telegramBackupKDFIDSize
	copy(header.RawKDFParams[:], envelope[offset:offset+telegramBackupKDFParamsSize])
	if header.KDFID == TelegramBackupKDFArgon2ID {
		header.KDFParams = TelegramBackupKDFParams{
			MemoryKiB:   binary.BigEndian.Uint32(header.RawKDFParams[0:4]),
			Iterations:  binary.BigEndian.Uint32(header.RawKDFParams[4:8]),
			Parallelism: header.RawKDFParams[8],
		}
	}
	offset += telegramBackupKDFParamsSize
	copy(header.Salt[:], envelope[offset:offset+telegramBackupSaltSize])
	offset += telegramBackupSaltSize
	copy(header.Nonce[:], envelope[offset:offset+telegramBackupNonceSize])
	return header, nil
}

func IsTelegramBackupEnvelope(data []byte) bool {
	return len(data) >= telegramBackupMagicSize && string(data[:telegramBackupMagicSize]) == TelegramBackupMagic
}

func buildTelegramBackupEnvelope(plaintext []byte, passphrase []byte, random io.Reader) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, common.NewError("missing_passphrase")
	}
	salt := make([]byte, telegramBackupSaltSize)
	nonce := make([]byte, telegramBackupNonceSize)
	defer zeroBytes(salt)
	defer zeroBytes(nonce)
	if _, err := io.ReadFull(random, salt); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(random, nonce); err != nil {
		return nil, err
	}

	params := telegramBackupDefaultKDFParams
	key := deriveTelegramBackupKey(passphrase, salt, params)
	defer zeroBytes(key)
	if len(key) != telegramBackupKeySize {
		return nil, common.NewError("invalid telegram backup key size")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, telegramBackupNonceSize)
	if err != nil {
		return nil, err
	}

	header := make([]byte, telegramBackupHeaderSize)
	copy(header[:telegramBackupMagicSize], []byte(TelegramBackupMagic))
	offset := telegramBackupMagicSize
	header[offset] = TelegramBackupEnvelopeVersion
	offset += telegramBackupVersionSize
	header[offset] = TelegramBackupKDFArgon2ID
	offset += telegramBackupKDFIDSize
	binary.BigEndian.PutUint32(header[offset:offset+4], params.MemoryKiB)
	binary.BigEndian.PutUint32(header[offset+4:offset+8], params.Iterations)
	header[offset+8] = params.Parallelism
	offset += telegramBackupKDFParamsSize
	copy(header[offset:offset+telegramBackupSaltSize], salt)
	offset += telegramBackupSaltSize
	copy(header[offset:offset+telegramBackupNonceSize], nonce)

	envelope := make([]byte, 0, len(header)+len(plaintext)+gcm.Overhead())
	envelope = append(envelope, header...)
	envelope = gcm.Seal(envelope, nonce, plaintext, header)
	return envelope, nil
}

func deriveTelegramBackupKey(passphrase []byte, salt []byte, params TelegramBackupKDFParams) []byte {
	return argon2.IDKey(passphrase, salt, params.Iterations, params.MemoryKiB, params.Parallelism, telegramBackupKeySize)
}

func validateTelegramBackupKDFParams(params TelegramBackupKDFParams) error {
	if params.MemoryKiB < telegramBackupArgon2MemoryKiB || params.MemoryKiB > 1024*1024 {
		return ErrTelegramBackupInvalidEnvelope
	}
	if params.Iterations == 0 || params.Iterations > 16 {
		return ErrTelegramBackupInvalidEnvelope
	}
	if params.Parallelism == 0 || params.Parallelism > 4 {
		return ErrTelegramBackupInvalidEnvelope
	}
	return nil
}
