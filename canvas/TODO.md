Bugs:

* box select tool switches back to arrow after selecting

* layers: active layer cannot be locked!

* diagonal constraint reshape with lower-right control and Control should constrain to aspect ratio,
but is instead constraining to diagonal movement, which is not the same!
    
* losing the Alt key in extended drawing sequences, so nodes are not smooth

* drawing a new element should be prioritized on mouse down -- only select on mouse up if drawing not started.

* duplicating solid element restores gradient in some cases

* grouping can change front / back order -- it is using the selection order instead of the existing order?

* more subtle logic about duplicating groups:
    + is it duplicating all the elements within the same group?
    + duplicating single element within group puts it in the group.. not what you expect.

* the first node in a path is getting abandoned and can cause crashing in delete.

* images in svg are not clipping to bounds -- renderer should do this in pimage.

