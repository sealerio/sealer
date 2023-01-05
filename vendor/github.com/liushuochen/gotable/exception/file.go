package exception

import "fmt"

type fileError struct {
	*baseError
	filename string
}

func createFileError(filename, message string) *fileError {
	err := &fileError{baseError: createBaseError(message), filename: filename}
	return err
}

func (err *fileError) Filename() string {
	return err.filename
}

type FileDoNotExistError struct {
	*fileError
}

func FileDoNotExist(path string) *FileDoNotExistError {
	message := fmt.Sprintf("file %s do not exist", path)
	err := &FileDoNotExistError{fileError: createFileError(path, message)}
	return err
}

type NotARegularCSVFileError struct {
	*fileError
}

func NotARegularCSVFile(path string) *NotARegularCSVFileError {
	message := fmt.Sprintf("not a regular csv file: %s", path)
	err := &NotARegularCSVFileError{fileError: createFileError(path, message)}
	return err
}

type NotARegularJSONFileError struct {
	*fileError
}

func NotARegularJSONFile(path string) *NotARegularJSONFileError {
	message := fmt.Sprintf("not a regular json file: %s", path)
	err := &NotARegularJSONFileError{fileError: createFileError(path, message)}
	return err
}

type UnSupportedFileTypeError struct {
	*fileError
}

func UnSupportedFileType(path string) *UnSupportedFileTypeError {
	message := fmt.Sprintf("Unsupported file type %s", path)
	err := &UnSupportedFileTypeError{
		fileError: createFileError(path, message),
	}
	return err
}

type NotGotableJSONFormatError struct {
	*fileError
}

func NotGotableJSONFormat(path string) *NotGotableJSONFormatError {
	message := fmt.Sprintf("json file %s is not a valid gotable json format", path)
	err := &NotGotableJSONFormatError{fileError: createFileError(path, message)}
	return err
}
