// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

// The Objective-C side of the package: a browser of camera devices of
// ImageCaptureCore, as iPhones, whose messages are on the main thread.

#include "imagecapture.h"
#import <Foundation/Foundation.h>
#import <ImageCaptureCore/ImageCaptureCore.h>
#include <stdlib.h>

#include "_cgo_export.h"

// onMain runs the given block on the main thread, where ImageCaptureCore
// sends its messages, and waits for it.
static void onMain(dispatch_block_t block) {
	if ([NSThread isMainThread]) {
		block();
	} else {
		dispatch_sync(dispatch_get_main_queue(), block);
	}
}

// icDevice is a camera device, as an iPhone, with its state.
@interface ICDev : NSObject
@property (strong) ICCameraDevice* device;
@property BOOL opened;
@property BOOL ready;
@property (strong) NSString* error;
@property BOOL locked;
@property (strong) NSArray<ICCameraFile*>* files;
@end
@implementation ICDev
@end

@interface ICDelegate : NSObject <ICDeviceBrowserDelegate, ICCameraDeviceDelegate>
@property (strong) ICDeviceBrowser* browser;
@property (strong) NSMutableArray<ICDev*>* devices;
@end

@implementation ICDelegate
- (ICDev*)devFor:(ICDevice*)device {
	for (ICDev* d in self.devices) {
		if (d.device == device) {
			return d;
		}
	}
	return nil;
}
- (void)deviceBrowser:(ICDeviceBrowser*)browser didAddDevice:(ICDevice*)device moreComing:(BOOL)moreComing {
	if (device.type & ICDeviceTypeCamera) {
		ICDev* d = [ICDev new];
		d.device = (ICCameraDevice*)device;
		device.delegate = self;
		[self.devices addObject:d];
	}
}
- (void)deviceBrowser:(ICDeviceBrowser*)browser didRemoveDevice:(ICDevice*)device moreGoing:(BOOL)moreGoing {
	ICDev* d = [self devFor:device];
	if (d != nil) {
		[self.devices removeObject:d];
	}
}
- (void)device:(ICDevice*)device didOpenSessionWithError:(NSError*)error {
	ICDev* d = [self devFor:device];
	if (error != nil) {
		// a locked phone opens once it is unlocked
		if (((ICCameraDevice*)device).isAccessRestrictedAppleDevice) {
			d.locked = YES;
			return;
		}
		d.error = error.localizedDescription;
		return;
	}
	d.locked = NO;
	d.opened = YES;
}
- (void)device:(ICDevice*)device didCloseSessionWithError:(NSError*)error {
	ICDev* d = [self devFor:device];
	d.opened = NO;
	d.ready = NO;
}
- (void)didRemoveDevice:(ICDevice*)device {}
- (void)deviceDidBecomeReadyWithCompleteContentCatalog:(ICCameraDevice*)camera {
	ICDev* d = [self devFor:camera];
	NSMutableArray* files = [NSMutableArray new];
	for (ICCameraItem* item in camera.mediaFiles) {
		if ([item isKindOfClass:[ICCameraFile class]]) {
			[files addObject:item];
		}
	}
	d.files = files;
	d.ready = YES;
}
- (void)cameraDevice:(ICCameraDevice*)camera didAddItems:(NSArray<ICCameraItem*>*)items {}
- (void)cameraDevice:(ICCameraDevice*)camera didRemoveItems:(NSArray<ICCameraItem*>*)items {}
- (void)cameraDevice:(ICCameraDevice*)camera didReceiveThumbnail:(CGImageRef)thumbnail forItem:(ICCameraItem*)item error:(NSError*)error {}
- (void)cameraDevice:(ICCameraDevice*)camera didReceiveMetadata:(NSDictionary*)metadata forItem:(ICCameraItem*)item error:(NSError*)error {}
- (void)cameraDevice:(ICCameraDevice*)camera didRenameItems:(NSArray<ICCameraItem*>*)items {}
- (void)cameraDeviceDidChangeCapability:(ICCameraDevice*)camera {}
- (void)cameraDevice:(ICCameraDevice*)camera didReceivePTPEvent:(NSData*)eventData {}
- (void)deviceDidBecomeReady:(ICDevice*)device {}
- (void)cameraDeviceDidRemoveAccessRestriction:(ICDevice*)device {
	ICDev* d = [self devFor:device];
	if (d != nil && d.locked) {
		d.locked = NO;
		d.error = nil;
		[device requestOpenSession];
	}
}
- (void)cameraDeviceDidEnableAccessRestriction:(ICDevice*)device {}
@end

static ICDelegate* theDelegate;

// icStart starts looking for camera devices, once.
void icStart() {
	onMain(^{
		if (theDelegate != nil) {
			return;
		}
		theDelegate = [ICDelegate new];
		theDelegate.devices = [NSMutableArray new];
		theDelegate.browser = [ICDeviceBrowser new];
		theDelegate.browser.delegate = theDelegate;
		theDelegate.browser.browsedDeviceTypeMask = ICDeviceTypeMaskCamera | ICDeviceLocationTypeMaskLocal;
		[theDelegate.browser start];
	});
}

int icNumDevices() {
	__block int n = 0;
	onMain(^{ n = (int)theDelegate.devices.count; });
	return n;
}

static ICDev* devAt(int i) {
	if (i < 0 || i >= theDelegate.devices.count) {
		return nil;
	}
	return theDelegate.devices[i];
}

// icDeviceName returns the name of the device (to be freed), or NULL.
char* icDeviceName(int i) {
	__block char* name = NULL;
	onMain(^{
		ICDev* d = devAt(i);
		if (d != nil && d.device.name != nil) {
			name = strdup(d.device.name.UTF8String);
		}
	});
	return name;
}

void icOpen(int i) {
	onMain(^{
		ICDev* d = devAt(i);
		if (d != nil && !d.opened) {
			d.error = nil;
			[d.device requestOpenSession];
		}
	});
}

void icClose(int i) {
	onMain(^{
		ICDev* d = devAt(i);
		if (d != nil && d.opened) {
			[d.device requestCloseSession];
		}
	});
}

// icState returns 0 if the device is gone, 1 while it is opening, 2 when
// it is ready, with all of its files, 3 for an error (to be freed in err),
// and 4 while it is locked, until it is unlocked.
int icState(int i, char** err) {
	__block int st = 0;
	onMain(^{
		ICDev* d = devAt(i);
		if (d == nil) {
			return;
		}
		if (d.error != nil) {
			*err = strdup(d.error.UTF8String);
			st = 3;
			return;
		}
		if (d.locked) {
			st = 4;
			return;
		}
		st = d.ready ? 2 : 1;
	});
	return st;
}


// icFiles returns the files of the device, of the count in n (to be freed,
// with their names).
icFile* icFiles(int i, int* n) {
	__block icFile* res = NULL;
	*n = 0;
	onMain(^{
		ICDev* d = devAt(i);
		if (d == nil || d.files == nil) {
			return;
		}
		int count = (int)d.files.count;
		res = calloc(count, sizeof(icFile));
		for (int k = 0; k < count; k++) {
			ICCameraFile* f = d.files[k];
			NSString* name = f.name != nil ? f.name : @"";
			res[k].name = strdup(name.UTF8String);
			res[k].size = f.fileSize;
			res[k].exifDate = f.exifCreationDate != nil ? f.exifCreationDate.timeIntervalSince1970 : 0;
			res[k].fileDate = f.fileCreationDate != nil ? f.fileCreationDate.timeIntervalSince1970 : 0;
			res[k].width = (int)f.width;
			res[k].height = (int)f.height;
			res[k].movie = [f.UTI isEqualToString:@"public.movie"] || f.duration > 0;
			res[k].duration = f.duration;
		}
		*n = count;
	});
	return res;
}

void icFreeFiles(icFile* files, int n) {
	for (int k = 0; k < n; k++) {
		free(files[k].name);
	}
	free(files);
}

// icDownload downloads the file at the given index of the device into the
// given directory as the given name, calling goDownloaded with the id.
void icDownload(int i, int k, char* dir, char* name, long long id) {
	NSString* dirS = [NSString stringWithUTF8String:dir];
	NSString* nameS = [NSString stringWithUTF8String:name];
	onMain(^{
		ICDev* d = devAt(i);
		if (d == nil || d.files == nil || k < 0 || k >= d.files.count) {
			goDownloaded(id, NULL, "the file is gone");
			return;
		}
		NSDictionary* opts = @{
			ICDownloadsDirectoryURL: [NSURL fileURLWithPath:dirS isDirectory:YES],
			ICSaveAsFilename: nameS,
			ICOverwrite: @YES,
		};
		[d.files[k] requestDownloadWithOptions:opts completion:^(NSString* filename, NSError* error) {
			if (error != nil) {
				goDownloaded(id, NULL, (char*)error.localizedDescription.UTF8String);
				return;
			}
			goDownloaded(id, (char*)(filename != nil ? filename.UTF8String : ""), NULL);
		}];
	});
}

// icRunLoop runs the run loop of the main thread for up to the given
// seconds, for programs without an event loop of their own, as tests,
// which must call it on the main thread.
void icRunLoop(double seconds) {
	CFRunLoopRunInMode(kCFRunLoopDefaultMode, seconds, false);
}
