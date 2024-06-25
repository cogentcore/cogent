// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/base/errors"
	_ "cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	_ "cogentcore.org/core/xyz/io/obj"
	"cogentcore.org/core/xyz/xyzcore"
)

func main() {
	b := core.NewBody("Cogent Shape")

	se := xyzcore.NewSceneEditor(b)
	se.UpdateWidget()
	sw := se.SceneWidget()
	sw.SelectionMode = xyzcore.Manipulable
	sc := se.SceneXYZ()

	// first, add lights, set camera
	sc.BackgroundColor = colors.ToUniform(colors.Scheme.Select.Container)
	xyz.NewAmbientLight(sc, "ambient", 0.3, xyz.DirectSun)

	dir := xyz.NewDirLight(sc, "dir", 1, xyz.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := xyz.NewPointLight(sc, "point", 1, xyz.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := xyz.NewSpotLight(sc, "spot", 1, xyz.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	sc.Camera.LookAt(math32.Vector3{}, math32.Vec3(0, 1, 0)) // defaults to looking at origin

	objgp := xyz.NewGroup(sc)

	curFn := "objs/airplane_prop_001.obj"
	// curFn := "objs/piano_005.obj"
	exts := ".obj,.dae,.gltf"

	errors.Log1(sc.OpenNewObj(curFn, objgp))

	b.AddAppBar(func(p *tree.Plan) {
		tree.Add(p, func(w *core.Button) {
			w.SetText("Open").SetIcon(icons.Open).
				SetTooltip("Open a 3D object file for viewing").
				OnClick(func(e events.Event) {
					core.FilePickerDialog(b, curFn, exts, "Open 3D Object", func(selFile string) {
						curFn = selFile
						objgp.DeleteChildren()
						sc.DeleteMeshes()
						sc.DeleteTextures()
						errors.Log1(sc.OpenNewObj(selFile, objgp))
						sc.SetCamera("default")
						sc.SetNeedsUpdate()
						se.NeedsRender()
					})
				})
		})
	})

	sc.SetNeedsConfig()
	b.RunMainWindow()
}
