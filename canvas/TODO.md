Bugs:

* drawing a new element should be prioritized on mouse down -- only select on mouse up if drawing not started.

* duplicating solid element restores gradient in some cases

* grouping can change front / back order

* more subtle logic about duplicating groups:
    + is it duplicating all the elements within the same group?
    + duplicating single element within group puts it in the group.. not what you expect.

* diagonal constraint reshape with lower-right control and Control should constrain to aspect ratio,
but is instead constraining to diagonal movement, which is not the same!
    
* images in svg are not clipping to bounds -- renderer should do this in pimage.

* still getting zero slider not sending change in setting a black color


