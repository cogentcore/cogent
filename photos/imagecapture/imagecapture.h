// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#pragma once

#include <stdlib.h>

typedef struct {
	char* name;
	long long size;
	double exifDate; // seconds since 1970, or 0
	double fileDate;
	int width, height;
	int movie;
	double duration;
} icFile;

void icStart(void);
int icNumDevices(void);
char* icDeviceName(int i);
void icOpen(int i);
void icClose(int i);
int icState(int i, char** err);
icFile* icFiles(int i, int* n);
void icFreeFiles(icFile* files, int n);
void icDownload(int i, int k, char* dir, char* name, long long id);
void icRunLoop(double seconds);
