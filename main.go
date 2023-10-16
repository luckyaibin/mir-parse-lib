package main

import (
	"fmt"
	"image"
	"image/png"
	"mirlib/log"
	"mirlib/mlibrary"
	"os"
	"path"
)

func main() {
	log.Tracef("hello")
	log.Infof("hello")
	log.Debugf("hello")
	log.Warnf("hello")
	log.Errorf("hello")

	//	charSel := mlibrary.NewMLibrary(`E:\exp\mir2-2022.06.12.00\Build\Client\Data\ChrSel.Lib`, 0)
	fileName := `D:\exp\mir2-2022.06.12.00\Build\Client\Data\Map\ShandaMir2\Tiles.Lib`
	charSel := mlibrary.NewMLibrary(fileName, 0)

	charSel.Initialize()
	bigImage := true
	if !bigImage {
		for i := int32(0); i < charSel.Count; i++ {
			charSel.CheckImage(int(i))
			charSel.SaveImage(int(i))
		}
	}
	// 保存成一张大图片
	if bigImage {
		name := fmt.Sprintf("%v.png", path.Base(fileName))
		bigTextureFile, _ := os.Create(name)

		//
		colCount := 20 // 每行20列
		rowCount := int(charSel.Count) / colCount

		w := 96
		h := 64
		//创建图片
		rect := image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{int(colCount * w), int(rowCount * h)},
		}
		bigTexture := image.NewRGBA(rect)
		for imageIndex := int(0); imageIndex < int(charSel.Count); imageIndex++ {
			atX := (imageIndex % colCount) * w
			atY := (imageIndex / colCount) * h
			charSel.SaveImageAt(bigTexture, atX, atY, imageIndex)
		}
		//bmp.Encode(bigTextureFile, bigTexture)
		png.Encode(bigTextureFile, bigTexture)
		bigTextureFile.Close()
	}

	return

	libs := mlibrary.InitLibrary(`E:\exp\mir2-2022.06.12.00\Build\Client\Data\Gate`, "00")

	for libIndex, lib := range libs {
		lib.Initialize()
		for imgIndex := int32(0); imgIndex < lib.Count; imgIndex++ {
			if libIndex == 6 && imgIndex == 8 {
				fmt.Print("")
			}
			ok := lib.CheckImage(int(imgIndex))
			if !ok {
				log.Errorf("libIndex %v imgIndex %v failed", libIndex, imgIndex)
			} else {
				log.Debugf("libIndex %v imgIndex %v ok", libIndex, imgIndex)
			}
			lib.SaveImage(int(imgIndex))
		}
	}
}
