# Web UI for Federated Storage Network

A simple web interface for testing file uploads and downloads in the Federated Storage Network.

## Features

- 🔐 User Authentication (Login/Register)
- 📤 File Upload with Drag & Drop
- 📁 File Listing and Management
- ⬇️ File Download
- 💰 Credit System Display
- 📱 Responsive Design

## Quick Start

### 1. Setup Test User

Before using the Web UI, set up a test user with credits:

```bash
./setup-test-user.sh
```

This creates:
- **Email**: `test@example.com`
- **Password**: `testpassword123`
- **Credits**: 10,000

### 2. Start the System

Start the coordinator and at least one storage node:

```bash
# Terminal 1: Start PostgreSQL
docker-compose up -d postgres

# Terminal 2: Start Coordinator
cd coordinator
go run cmd/api/main.go

# Terminal 3: Start Storage Node
cd storage-node
go run cmd/storage-node/main.go init --name "Test Node" --coordinator-url http://localhost:8080
go run cmd/storage-node/main.go start
```

### 3. Access the Web UI

Open your browser and go to:

```
http://localhost:8080/web/
```

Or simply:

```
http://localhost:8080/
```

### 4. Login and Test

1. Use the pre-filled test user credentials:
   - Email: `test@example.com`
   - Password: `testpassword123`

2. Click "Login"

3. Upload files by:
   - Dragging and dropping files onto the upload area
   - Clicking the upload area to select files

4. View your uploaded files in the file list

5. Download files by clicking the "Download" button

## Web UI Instructions (Shown on Page)

The Web UI includes inline instructions:

1. **Make sure the coordinator is running**
   ```bash
   cd coordinator && go run cmd/api/main.go
   ```

2. **At least one storage node should be registered and running**

3. **Use the test user credentials below, or register a new user**

4. **Login to get your access token**

5. **Upload files** (they'll be encrypted and distributed across storage nodes)

6. **View your files and download them back**

## Architecture

The Web UI is a simple static HTML/CSS/JS application served by the coordinator:

```
coordinator/web/static/
├── index.html    # Main UI with instructions
└── app.js        # JavaScript for API calls
```

The UI communicates directly with the coordinator's REST API:
- `POST /api/v1/auth/login` - Authentication
- `POST /api/v1/files/upload/initiate` - Start upload
- `POST /api/v1/files/upload/{id}/chunk` - Upload chunks
- `POST /api/v1/files/upload/{id}/complete` - Complete upload
- `GET /api/v1/files` - List files
- `GET /api/v1/files/{id}/download` - Download file
- `DELETE /api/v1/files/{id}` - Delete file

## Browser Compatibility

- Chrome 80+
- Firefox 75+
- Safari 13+
- Edge 80+

All modern browsers that support:
- ES6 JavaScript
- Fetch API
- File API

## Troubleshooting

### "Cannot connect to server"
- Make sure the coordinator is running on port 8080
- Check that PostgreSQL is running
- Verify there are no firewall blocking connections

### "Login failed"
- Run `./setup-test-user.sh` to create the test user
- Check the coordinator logs for errors
- Make sure the database migrations ran successfully

### "Upload failed"
- Ensure you have sufficient credits (check the credits display)
- Make sure at least one storage node is registered and running
- Check browser console for JavaScript errors
- Verify the file isn't too large for your available credits

### "CORS errors in browser console"
- The coordinator has CORS enabled by default
- If you're accessing from a different origin, update the CORS settings in `cmd/api/main.go`

## Customization

You can customize the UI by editing:
- `coordinator/web/static/index.html` - Structure and styling
- `coordinator/web/static/app.js` - JavaScript logic and API calls

The UI uses vanilla JavaScript with no external dependencies, making it easy to modify.

## Security Notes

- The Web UI is for testing purposes only
- Files are encrypted with AES-256-GCM before storage
- JWT tokens are stored in browser localStorage (cleared on logout)
- Always use HTTPS in production (not included in MVP)

## Screenshots

The UI includes:
- Clean gradient header with app title
- Step-by-step instructions panel
- Test user credentials box
- Login/Register form
- Drag-and-drop file upload area
- Progress bar during upload
- File list with download/delete buttons
- Credit balance display
- Responsive mobile-friendly design

## API Integration

The Web UI demonstrates how to integrate with the Federated Storage Network API:

```javascript
// Login
const response = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password })
});
const { token } = await response.json();

// Upload File (3-step process)
// 1. Initiate
// 2. Upload chunks
// 3. Complete

// List Files
const files = await fetch('/api/v1/files', {
    headers: { 'Authorization': `Bearer ${token}` }
});

// Download File
await fetch(`/api/v1/files/${fileId}/download`, {
    headers: { 'Authorization': `Bearer ${token}` }
});
```

See `app.js` for complete implementation details.

## Future Enhancements

- [ ] Folder support
- [ ] File sharing links
- [ ] Thumbnail previews for images
- [ ] Upload resume capability
- [ ] Dark mode toggle
- [ ] Multiple file upload queue
- [ ] Progress percentage display
- [ ] Drag-and-drop folder upload