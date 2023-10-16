package mlibrary

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"io/ioutil"
	"mirlib/log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/image/bmp"
)

func InitLibrary(directory string, toStringValue string) map[int64]*MLibrary {
	libraries := make(map[int64]*MLibrary)
	if !PathExists(directory) {
		os.Create(directory)
	}
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		log.Fatalf("read path %v ", err)
	}
	filtered := make([]fs.FileInfo, 0)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".Lib") {
			filtered = append(filtered, f)
		}
	}
	if len(filtered) > 0 {
		reg, err := regexp.CompilePOSIX(`[0-9]+`)
		if err != nil {
			log.Errorf("regexp.Compile error : %v", err)
		}
		for _, f := range filtered {
			name := f.Name()

			indices := reg.FindStringSubmatch(name)
			if len(indices) == 0 {
				log.Errorf("file name invalid : %s", name)
				continue
			}
			index, err := strconv.Atoi(indices[0])
			if err != nil {
				log.Errorf("%v", err)
				continue
			}
			libraries[int64(index)] = NewMLibrary(path.Clean(directory+"/"+f.Name()), index)
			log.Debugf("load %v ok...", f.Name())
		}
	}
	return libraries
}

func NewMLibrary(filename string, libIndex int) *MLibrary {
	m := &MLibrary{
		libIndex: libIndex,
		filename: filename,
		_frames:  make(map[byte]*Frame),
	}
	return m
}

type MLibrary struct {
	filename    string
	libIndex    int
	file        *os.File
	reader      *bufio.Reader
	initialized bool
	Count       int32
	indexList   []int32
	images      []*MImage

	_frames map[byte]*Frame
}

/*
一个MLibrary代表一个xxx.Lib文件，里面有图片，有frame
4字节版本号
*/
func (m *MLibrary) Initialize() {
	m.initialized = true
	if !PathExists(m.filename) {
		return
	}

	f, err := os.Open(m.filename)
	if err != nil {
		log.Fatalf("open file error %v", err)
		return
	}
	m.file = f

	reader := bufio.NewReader(m.file)
	m.reader = reader
	var currentVersion int32
	binary.Read(reader, binary.LittleEndian, &currentVersion)
	if currentVersion < 2 {
		log.Fatalf("version less 2")
		return
	}
	binary.Read(reader, binary.LittleEndian, &m.Count)

	var frameSeek int32 = 0
	if currentVersion >= 3 {
		binary.Read(reader, binary.LittleEndian, &frameSeek)
	}
	m.images = make([]*MImage, m.Count)
	m.indexList = make([]int32, m.Count)
	for i := int32(0); i < m.Count; i++ {
		index := int32(0)
		binary.Read(reader, binary.LittleEndian, &index)
		m.indexList[i] = index
	}
	if currentVersion >= 3 {
		log.Debugf("%v version is %v", m.filename, currentVersion)
		m.file.Seek(int64(frameSeek), 0)
		reader.Reset(m.file)

		var frameCount int32
		binary.Read(reader, binary.LittleEndian, &frameCount)
		if frameCount > 0 {
			for i := int32(0); i < frameCount; i++ {
				frameType := byte(0)
				binary.Read(reader, binary.LittleEndian, &frameType)
				frame := NewFrameWithReader(reader)
				m._frames[frameType] = frame
			}
		}
	}
}

func (m *MLibrary) CheckImage(index int) bool {
	if !m.initialized {
		m.Initialize()
	}
	if index > len(m.images) || index < 0 {
		return false
	}
	if m.images[index] == nil {
		offsetOfIndex := m.indexList[index]
		//跳到对应位置读取图片信息
		m.reader.Reset(m.file)
		m.file.Seek(int64(offsetOfIndex), 0)
		img := NewMImage(m.reader)
		m.images[index] = img
	}

	mi := m.images[index]
	if !mi.TextureValid {
		if mi.Width == 0 || mi.Height == 0 {
			return false
		}
		// 跳到对应位置读取图片数据 17是 Width(2) + Height(2) + X(2) + Y(2) + ShadowX(2) + ShadowY(2) + Shadow(1) + Length(4)
		m.file.Seek(int64(m.indexList[index])+17, 0)
		m.reader.Reset(m.file)

		mi.CreateTexture(m.reader)
	}
	return true
}

func (m *MLibrary) SaveImage(index int) bool {
	if m.CheckImage(index) {
		img := m.images[index]
		if img != nil {
			name := fmt.Sprintf("%d_%d.bmp", m.libIndex, index)
			f, err := os.Create(name)
			if err != nil {
				log.Fatalf("create file %v failed %v", name, err)
			}

			//创建图片
			rect := image.Rectangle{
				Min: image.Point{0, 0},
				Max: image.Point{int(img.Width), int(img.Height)},
			}

			texture := image.NewRGBA(rect)
			bytes := img.TextureData
			if img.Height < 0 {
				log.Fatalf("invalid height %v %v", m.libIndex, index)
			}
			for h := int(0); h < int(img.Height); h++ {
				for w := int(0); w < int(img.Width); w++ {
					pixAt := (h * 4 * int(img.Width)) + w*4 //bgra获取像素位置和颜色，写入Image对应位置
					if pixAt < 0 {
						fmt.Print("")
					}
					a := bytes[pixAt+3]
					r := bytes[pixAt+2]
					g := bytes[pixAt+1]
					b := bytes[pixAt+0]
					texture.SetRGBA(int(w), int(h), color.RGBA{R: r, G: g, B: b, A: a})
				}
			}
			bmp.Encode(f, texture)
			//f.Write(image.Data.Bytes())
			f.Close()
			return true
		}
	}
	return false
}
func (m *MLibrary) SaveImageAt(texture *image.RGBA, x, y int, index int) bool {
	if m.CheckImage(index) {
		img := m.images[index]
		if img != nil {
			bytes := img.TextureData
			if img.Height < 0 {
				log.Fatalf("invalid height %v %v", m.libIndex, index)
			}
			for h := int(0); h < int(img.Height); h++ {
				for w := int(0); w < int(img.Width); w++ {
					pixAt := (h * 4 * int(img.Width)) + w*4 //bgra获取像素位置和颜色，写入Image对应位置
					if pixAt < 0 {
						fmt.Print("")
					}
					a := bytes[pixAt+3]
					r := bytes[pixAt+2]
					g := bytes[pixAt+1]
					b := bytes[pixAt+0]
					texture.SetRGBA(int(w)+x, int(h)+y, color.RGBA{R: r, G: g, B: b, A: a})
				}
			}
			return true
		}
	}
	return false
}

func NewMImage(reader *bufio.Reader) *MImage {
	m := &MImage{}
	binary.Read(reader, binary.LittleEndian, &m.Width)
	binary.Read(reader, binary.LittleEndian, &m.Height)
	binary.Read(reader, binary.LittleEndian, &m.X)
	binary.Read(reader, binary.LittleEndian, &m.Y)
	binary.Read(reader, binary.LittleEndian, &m.ShadowX)
	binary.Read(reader, binary.LittleEndian, &m.ShadowY)
	binary.Read(reader, binary.LittleEndian, &m.Shadow)
	binary.Read(reader, binary.LittleEndian, &m.Length)
	if m.Shadow>>7 == 1 {
		m.HasMask = true
	}
	//check if there's a second layer and read it
	if m.HasMask {
		//跳过图片的长度,暂时这样做
		content := make([]byte, m.Length)
		reader.Read(content)
		binary.Read(reader, binary.LittleEndian, &m.MaskWidth)
		binary.Read(reader, binary.LittleEndian, &m.MaskHeight)
		binary.Read(reader, binary.LittleEndian, &m.MaskX)
		binary.Read(reader, binary.LittleEndian, &m.MaskY)
		binary.Read(reader, binary.LittleEndian, &m.MaskLength)
	}
	return m
}

type MImage struct {
	//以下部分17字节长
	Width   int16
	Height  int16
	X       int16
	Y       int16
	ShadowX int16
	ShadowY int16
	Shadow  byte
	Length  int32

	TextureData []byte

	HasMask    bool
	MaskWidth  int16
	MaskHeight int16
	MaskX      int16
	MaskY      int16
	MaskLength int32

	MaskTextureData []byte

	TextureValid bool
}

func (m *MImage) CreateTexture(reader *bufio.Reader) {
	Data := make([]byte, m.Length)
	reader.Read(Data)
	gzipReader, err := gzip.NewReader(bytes.NewBuffer(Data))
	if err != nil {
		log.Fatalf("CreateTexture gzip new reader failed %v", err)
	}
	log.Debugf("texture %#v", m)

	m.TextureData, err = ioutil.ReadAll(gzipReader)
	if err != nil {
		log.Fatalf("CreateTexture ioutil.ReadAll(gzipReader) failed %v", err)
	}
	if m.HasMask {
		//跳过12字节
		reader.Read(make([]byte, 12))
		MaskData := make([]byte, m.MaskLength)
		reader.Read(MaskData)
		gzipReader, err = gzip.NewReader(bytes.NewBuffer(Data))
		if err != nil {
			log.Fatalf("CreateTexture Mask gzip new reader failed %v", err)
		}
		m.MaskTextureData, err = ioutil.ReadAll(gzipReader)
		if err != nil {
			log.Fatalf("CreateTexture Mask ioutil.ReadAll(gzipReader) failed %v", err)
		}
	}
	m.TextureValid = true
}

// func (m *MImage) CreateTexture(reader *bufio.Reader) {
// 	Data := make([]byte, m.Length)
// 	reader.Read(Data)

// 	m.Data = bytes.NewBuffer(Data)
// 	if m.HasMask {
// 		//跳过12字节
// 		reader.Read(make([]byte, 12))
// 		MaskData := make([]byte, m.MaskLength)
// 		reader.Read(MaskData)
// 		m.MaskData = bytes.NewBuffer(MaskData)
// 	}
// 	m.TextureValid = true
// }

func NewFrame(start, count, skip, interval int32, effectstart, effectcount, effectskip, effectinterval int32) *Frame {
	return &Frame{
		Start:    start,
		Count:    count,
		Skip:     skip,
		Interval: interval,

		EffectStart:    effectstart,
		EffectCount:    effectcount,
		EffectSkip:     effectskip,
		EffectInterval: effectinterval,
	}
}

func NewFrameWithReader(reader *bufio.Reader) *Frame {
	f := &Frame{}
	binary.Read(reader, binary.LittleEndian, &f.Start)
	binary.Read(reader, binary.LittleEndian, &f.Count)
	binary.Read(reader, binary.LittleEndian, &f.Skip)
	binary.Read(reader, binary.LittleEndian, &f.Interval)
	binary.Read(reader, binary.LittleEndian, &f.EffectStart)
	binary.Read(reader, binary.LittleEndian, &f.EffectCount)
	binary.Read(reader, binary.LittleEndian, &f.EffectSkip)
	binary.Read(reader, binary.LittleEndian, &f.EffectInterval)
	binary.Read(reader, binary.LittleEndian, &f.Reverse)
	binary.Read(reader, binary.LittleEndian, &f.Blend)
	return f
}

type Frame struct {
	Start, Count, Skip                   int32
	EffectStart, EffectCount, EffectSkip int32
	Interval, EffectInterval             int32
	Reverse, Blend                       bool
}

func (f *Frame) OffSet() int32 {
	return f.Count + f.Skip
}

func (f *Frame) EffectOffSet() int32 {
	return f.EffectCount + f.EffectSkip
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
