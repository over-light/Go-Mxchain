package core

import (
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml"
)

const fileModeUserReadWrite = 0600

// OpenFile method opens the file from given path - does not close the file
func OpenFile(relativePath string) (*os.File, error) {
	path, err := filepath.Abs(relativePath)
	if err != nil {
		log.Warn("cannot create absolute path for the provided file", "error", err.Error())
		return nil, err
	}
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	return f, nil
}

// LoadTomlFile method to open and decode toml file
func LoadTomlFile(dest interface{}, relativePath string) error {
	f, err := OpenFile(relativePath)
	if err != nil {
		return err
	}

	defer func() {
		err = f.Close()
		if err != nil {
			log.Warn("cannot close file", "error", err.Error())
		}
	}()

	return toml.NewDecoder(f).Decode(dest)
}

// LoadTomlFileToMap opens and decodes a toml file as a map[string]interface{}
func LoadTomlFileToMap(relativePath string) (map[string]interface{}, error) {
	f, err := OpenFile(relativePath)
	if err != nil {
		return nil, err
	}

	fileinfo, err := f.Stat()
	if err != nil {
		log.Error("cannot stat file:", err.Error())
		return nil, err
	}

	filesize := fileinfo.Size()
	buffer := make([]byte, filesize)

	_, err = f.Read(buffer)
	if err != nil {
		log.Error("cannot read from file:", err.Error())
		return nil, err
	}

	defer func() {
		err = f.Close()
		if err != nil {
			log.Error("cannot close file: ", err.Error())
		}
	}()

	loadedTree, err := toml.Load(string(buffer))
	if err != nil {
		log.Error("cannot interpret file contents as toml:", err.Error())
		return nil, err
	}

	loadedMap := loadedTree.ToMap()

	return loadedMap, nil
}

// LoadJsonFile method to open and decode json file
func LoadJsonFile(dest interface{}, relativePath string) error {
	f, err := OpenFile(relativePath)
	if err != nil {
		return err
	}

	defer func() {
		err = f.Close()
		if err != nil {
			log.Warn("cannot close file", "error", err.Error())
		}
	}()

	return json.NewDecoder(f).Decode(dest)
}

// CreateFile opens or creates a file relative to the default path
func CreateFile(prefix string, subfolder string, fileExtension string) (*os.File, error) {
	absPath, err := filepath.Abs(subfolder)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(absPath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	fileName := time.Now().Format("2006-02-01-15-04-05")
	if prefix != "" {
		fileName = prefix + "-" + fileName
	}

	return os.OpenFile(
		filepath.Join(absPath, fileName+"."+fileExtension),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		fileModeUserReadWrite)
}

// LoadSkFromPemFile loads the secret key bytes stored in the file
func LoadSkFromPemFile(relativePath string, skIndex int) ([]byte, error) {
	if skIndex < 0 {
		return nil, ErrInvalidIndex
	}

	file, err := OpenFile(relativePath)
	if err != nil {
		return nil, err
	}

	defer func() {
		cerr := file.Close()
		log.LogIfError(cerr)
	}()

	buff, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if len(buff) == 0 {
		return nil, ErrEmptyFile
	}

	var blkRecovered *pem.Block

	for i := 0; i <= skIndex; i++ {
		if len(buff) == 0 {
			//less private keys present in the file than required
			return nil, ErrInvalidIndex
		}

		blkRecovered, buff = pem.Decode(buff)
		if blkRecovered == nil {
			return nil, ErrPemFileIsInvalid
		}
	}

	return blkRecovered.Bytes, nil
}

// SaveSkToPemFile saves secret key bytes in the file
func SaveSkToPemFile(file *os.File, identifier string, skBytes []byte) error {
	if file == nil {
		return ErrNilFile
	}

	blk := pem.Block{
		Type:  "PRIVATE KEY for " + identifier,
		Bytes: skBytes,
	}

	return pem.Encode(file, &blk)
}
