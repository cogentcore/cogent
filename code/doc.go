// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gide is the top-level repository for the Gide Go-IDE based on
the GoGi GUI framework.

The following sub-packages provide all the code:

* gide: is the main codebase for the IDE, in library form which can be
used in different contexts.

* gidev: provides the GideView that is the actual GUI widget for the full IDE

* piv: is the GUI viewer for the GoPi interactive parser -- it is here because
it incorporates the Gide toolkit and Gide can run it as a sub-editor.

* cmd: is where the actual command tools are built.
*/
package gide
