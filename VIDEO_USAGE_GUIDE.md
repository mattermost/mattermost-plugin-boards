# Video Block Usage Guide

## Overview

The Mattermost Boards plugin includes a fully-featured video block that supports:
- 🎥 YouTube videos
- 📁 Google Drive videos
- 💾 Uploaded video files
- 🖥️ Fullscreen playback with controls

## How to Add Videos

### Method 1: Using the Slash Command

1. In any card, type `/video`
2. You'll see two options:
   - **Paste URL**: For YouTube or Google Drive links
   - **Upload File**: For local video files

### Method 2: From the Content Menu

1. Click the "+" button in a card
2. Select "Video" from the content type menu
3. Choose your input method

## Supported Video Sources

### 1. YouTube Videos

**Supported URL formats:**
- `https://youtube.com/watch?v=VIDEO_ID`
- `https://youtu.be/VIDEO_ID`
- `https://youtube.com/embed/VIDEO_ID`

**Example:**
```
/video
Paste: https://youtube.com/watch?v=dQw4w9WgXcQ
Click "Add"
```

**What you'll see:**
- YouTube thumbnail preview
- Play button overlay
- "YouTube" label below the video

**Fullscreen mode:**
- Click the thumbnail to open fullscreen
- YouTube player with full controls
- Auto-play enabled

### 2. Google Drive Videos

**Supported URL format:**
- `https://drive.google.com/file/d/FILE_ID/view`

**Example:**
```
/video
Paste: https://drive.google.com/file/d/1ABC123xyz/view
Click "Add"
```

**What you'll see:**
- Google Drive icon with gradient background
- Play button overlay
- "Google Drive" label below the video

**Fullscreen mode:**
- Click the icon to open fullscreen
- Google Drive preview player
- Embedded controls

### 3. Uploaded Video Files

**Supported formats:**
- Any format accepted by HTML5 video element
- Common formats: MP4, WebM, OGG, MOV

**Example:**
```
/video
Click "Upload File"
Select video from your computer
```

**What you'll see:**
- Video preview (first frame)
- Play button overlay
- Filename label below the video

**Fullscreen mode:**
- Click the preview to open fullscreen
- HTML5 video player with full controls
- Play/pause, seek, volume controls

## User Interface Features

### Preview Card
- **16:9 Aspect Ratio**: Consistent sizing for all videos
- **Play Button Overlay**: Appears on hover
- **Metadata Display**: Shows source type or filename
- **Keyboard Accessible**: Tab to focus, Enter/Space to play

### Fullscreen Viewer
- **Dark Backdrop**: 90% opacity black background
- **Close Button**: Top-right corner
- **Keyboard Shortcuts**:
  - `ESC` - Close viewer
  - `Space` - Play/pause (for uploaded files)
- **Click Outside**: Click backdrop to close
- **Responsive**: Adapts to mobile screens

## Keyboard Navigation

1. **Tab** to focus on video preview
2. **Enter** or **Space** to open fullscreen
3. **ESC** to close fullscreen
4. **Tab** to navigate to close button

## Mobile Support

- Touch-friendly play button
- Responsive sizing
- Native video controls on mobile devices
- Fullscreen mode optimized for small screens

## Examples

### Example 1: Adding a Tutorial Video
```
1. Create a new card "Product Tutorial"
2. Type /video
3. Paste: https://youtube.com/watch?v=abc123
4. Click "Add"
5. Video thumbnail appears in the card
```

### Example 2: Sharing a Team Recording
```
1. Create a card "Team Meeting - Jan 2026"
2. Type /video
3. Click "Upload File"
4. Select "meeting-recording.mp4"
5. Video preview appears in the card
```

### Example 3: Embedding a Google Drive Video
```
1. Create a card "Training Materials"
2. Type /video
3. Paste: https://drive.google.com/file/d/1XYZ789/view
4. Click "Add"
5. Google Drive icon appears in the card
```

## Tips & Best Practices

1. **YouTube Thumbnails**: Automatically fetched from YouTube's servers
2. **File Size**: Check your Mattermost server's max file size setting
3. **Privacy**: Google Drive videos require appropriate sharing permissions
4. **Performance**: Embedded videos don't load until clicked (lazy loading)
5. **Accessibility**: All videos include ARIA labels for screen readers

## Troubleshooting

### Video URL Not Recognized
- Ensure the URL matches supported formats
- Check for typos in the URL
- Try copying the URL directly from the browser address bar

### Google Drive Video Won't Play
- Verify the file is shared with "Anyone with the link"
- Check that the file is a video (not a folder or document)
- Ensure the URL includes `/file/d/FILE_ID/view`

### Uploaded Video Won't Play
- Check file format compatibility
- Verify file size is within server limits
- Try converting to MP4 format

## Technical Details

### Supported Video Formats (Uploaded Files)
- MP4 (H.264 codec recommended)
- WebM
- OGG
- MOV (may require conversion)

### Browser Compatibility
- Chrome/Edge: Full support
- Firefox: Full support
- Safari: Full support
- Mobile browsers: Full support

## Developer Information

For developers working with the video block:

**Files:**
- `webapp/src/components/blocksEditor/blocks/video/index.tsx`
- `webapp/src/components/videoViewer/videoViewer.tsx`
- `webapp/src/components/blocksEditor/blocks/video/video.scss`
- `webapp/src/components/videoViewer/videoViewer.scss`

**Tests:**
```bash
cd webapp
npm test -- --testPathPattern=video
```

**Development Mode:**
```bash
cd webapp
npm run deveditor
# Open http://localhost:9000/editor.html
```

