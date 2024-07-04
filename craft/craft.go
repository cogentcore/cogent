// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/base/errors"
	_ "cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	_ "cogentcore.org/core/xyz/io/obj"
	"cogentcore.org/core/xyz/xyzcore"
)

func main() {
	b := core.NewBody("Cogent Craft")

	se := xyzcore.NewSceneEditor(b)
	se.UpdateWidget()
	sw := se.SceneWidget()
	sw.SelectionMode = xyzcore.Manipulable
	sc := se.SceneXYZ()

	se.Styler(func(s *styles.Style) {
		sc.Background = colors.Scheme.Select.Container
	})
	xyz.NewAmbientLight(sc, "ambient", 0.3, xyz.DirectSun)

	dir := xyz.NewDirLight(sc, "dir", 1, xyz.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := xyz.NewPointLight(sc, "point", 1, xyz.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := xyz.NewSpotLight(sc, "spot", 1, xyz.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	sc.Camera.LookAt(math32.Vector3{}, math32.Vec3(0, 1, 0)) // defaults to looking at origin

	objgp := xyz.NewGroup(sc)

	currentFile := "objs/airplane_prop_001.obj"
	errors.Log1(sc.OpenNewObj(currentFile, objgp))

	b.AddAppBar(func(p *tree.Plan) {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(func(file core.Filename) {
				currentFile = string(file)
				objgp.DeleteChildren()
				sc.DeleteMeshes()
				sc.DeleteTextures()
				errors.Log1(sc.OpenNewObj(string(file), objgp))
				sc.SetCamera("default")
				sc.SetNeedsUpdate()
				se.NeedsRender()
			})
			w.Args[0].SetTag(`ext:".obj,.dae,.gltf"`)
			w.SetText("Open").SetIcon(icons.Open).SetTooltip("Open a 3D object file for viewing")
		})
	})

	sc.SetNeedsConfig()
	b.RunMainWindow()
}
