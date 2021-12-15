package tool

import (
	"autobuildrobot/log"
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/astaxie/beego/cache"
	"github.com/axgle/mahonia"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

type ExecCommandFunc func(result string)

//阻塞式的执行外部shell命令的函数,等待执行完毕并返回标准输出
func Exec_shell(cmdName,s string) (string, error) {
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command(cmdName, "-c", s)

	//读取io.Writer类型的cmd.Stdout，再通过bytes.Buffer(缓冲byte类型的缓冲器)将byte类型转化为string类型(out.String():这是bytes类型提供的接口)
	var out bytes.Buffer
	cmd.Stdout = &out

	//Run执行c包含的命令，并阻塞直到完成。  这里stdout被取出，cmd.Wait()无法正确获取stdin,stdout,stderr，则阻塞在那了
	err := cmd.Run()
	var enc mahonia.Decoder
	if runtime.GOOS == "windows" {
		enc = mahonia.NewDecoder("gbk")
	}else{
		enc = mahonia.NewDecoder("utf-8")
	}

	return enc.ConvertString(out.String()), err
}

//阻塞式的执行外部shell命令的函数,标准输出的逐行实时进行处理的
func ExecCommand(cmdName,command string,execCommandFunc ExecCommandFunc ) bool {
	log.Info("开始执行：" + command)
	cmd := exec.Command(cmdName, "-c", command)

	//StdoutPipe方法返回一个在命令Start后与命令标准输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error(err)
		return false
	}
	cmd.Start()

	//创建一个流来读取管道内内容，这里逻辑是通过一行一行的读取的
	reader := bufio.NewReader(stdout)
	var enc mahonia.Decoder
	if runtime.GOOS == "windows" {
		enc = mahonia.NewDecoder("gbk")
	}else{
		enc = mahonia.NewDecoder("utf-8")
	}
	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			log.Error("err2 != nil || io.EOF == err2")
			break
		}
		if line == "\r\n"{
			continue
		}
		temp := enc.ConvertString(line)
		execCommandFunc(temp)
	}

	//阻塞直到该命令执行完成，该命令必须是被Start方法开始执行的
	cmd.Wait()
	execCommandFunc("执行完毕！")
	return true
}

//发送http请求
func Http(requestType,url,content string)(error, []byte){
	//创建一个请求
	result :=""
	req, err := http.NewRequest(requestType, url, strings.NewReader(content))
	if err != nil {
		result = "发送http请求异常：" + err.Error()
		log.Error(result)
		return errors.New(result),nil
	}

	client := &http.Client{}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		result = "发送http请求失败：" + err.Error()
		log.Error(result)
		return errors.New(result),nil
	}
	body,err:=ioutil.ReadAll(resp.Body)
	return nil,body
}

//序列化json
func MarshalJson(jsonData interface{})string{
	data, _ := json.Marshal(jsonData)
	return string(data)
}

//反序列化json
func UnmarshJson(jsonData []byte,data interface{})error{
	return json.Unmarshal([]byte(jsonData), data)
}

var gobDataFilePath = "gobData" //gob文件夹名字
//读取gob文件
func ReadGobFile(fileName string,data interface{}){
	var dataFile = path.Join(gobDataFilePath, fileName)
	_, err := os.Stat(dataFile)
	if err == nil {
		content, err := ioutil.ReadFile(dataFile)
		if err != nil {
			log.Error("读取用户数据配置文件失败：" + err.Error())
			return
		}
		buf := bytes.NewBuffer(content)
		dec := gob.NewDecoder(buf)
		dec.Decode(data)
	} else {
		_, existPath := os.Stat(gobDataFilePath)
		if nil != existPath {
			os.MkdirAll(gobDataFilePath, os.ModePerm)
		}
	}
}

//保存gob数据
func SaveGobFile(fileName string,_data interface{})(result string) {
	//编码并存储
	data, errEncodeUser := cache.GobEncode(_data)
	if nil != errEncodeUser {
		result = "编码用户数据失败：" + errEncodeUser.Error()
		log.Error(result)
		return
	}
	fileObj, err := os.Create(path.Join(gobDataFilePath, fileName))
	if err != nil {
		result = "获取用户文件失败：" + err.Error()
		return
	}
	writer := bufio.NewWriter(fileObj)
	defer writer.Flush()
	writer.Write(data)
	return
}

//判断文件是否存在
func CheckFileIsExist(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

// 拷贝文件
func CopyFile(dstName, srcName string) (written int64, err error) {
	if CheckFileIsExist(dstName) {
		os.Remove(dstName)
	}
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

//计算文件md5值
func CalcMd5(filePath string)string{
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ""
	}
	md5 := md5.New()
	md5.Write(data)
	return hex.EncodeToString(md5.Sum(nil))
}