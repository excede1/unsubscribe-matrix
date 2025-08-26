# Customer.io Email Management Application

## 🚀 Quick Start Guide

### 1. **Start the Application**
```bash
# Build and run
go build -o main .
./main
```

### 2. **Access the Admin Dashboard**
- **URL**: `http://localhost:3000/results`
- **Username**: `morgan@excede.com.au`
- **Password**: `hdhgh&-TRFTuyVUYfftyfgh`

### 3. **Customer Email Management**
- **Customer Interface**: `http://localhost:3000/?email=customer@example.com`
- Customers can pause emails, switch to international list, or unsubscribe

### 4. **Admin Features**
- **View Records**: See all email processing activity
- **Download CSV**: Click download buttons under each action summary
- **Clear Records**: Click the page title to reveal the clear button

---

## 📋 What This Application Does

This is a comprehensive email preference management system for Customer.io that provides:

### **Customer-Facing Features**
- **Pause Emails**: Temporarily pause sale emails (sets `paused: true`)
- **International List**: Move to Australian/International list (removes BBUS relationship, adds BBAU)
- **Unsubscribe**: Permanently unsubscribe from all communications (sets `unsubscribed: true`)

### **Admin Dashboard Features**
- **Real-time Analytics**: View summary counts for each action type
- **Detailed Records**: See all customer actions with timestamps in Sydney timezone
- **CSV Export**: Download filtered records by action type (PAUSE, BBAU, UNSUBSCRIBE)
- **Database Management**: Clear all records with confirmation (hidden feature)

---

## ⚙️ Configuration

### **Environment Variables (.env file)**
```env
# Customer.io Track API credentials
CUSTOMERIO_SITE_ID=your_site_id_here
CUSTOMERIO_API_KEY=your_api_key_here

# Admin dashboard credentials
ADMIN_USERNAME=morgan@excede.com.au
ADMIN_PASSWORD=hdhgh&-TRFTuyVUYfftyfgh

# Optional: Server port (default: 3000)
PORT=3000
```

### **Required Setup**
1. **Customer.io Account**: Active account with Track API access
2. **API Credentials**: Site ID and API Key from Customer.io dashboard
3. **Go Environment**: Go 1.24.2 or higher

---

## 🔧 Installation & Setup

### **1. Clone and Install**
```bash
git clone <repository-url>
cd customerio-email-management
go mod download
```

### **2. Configure Environment**
```bash
# Copy and edit environment file
cp .env.example .env
# Edit .env with your Customer.io credentials
```

### **3. Build and Run**
```bash
go build -o main .
./main
```

### **4. Verify Installation**
```bash
# Test health check
curl http://localhost:3000/ping
# Response: pong

# Test customer interface
curl "http://localhost:3000/?email=test@example.com"
```

---

## 📊 Admin Dashboard Usage

### **Accessing the Dashboard**
1. Navigate to `http://localhost:3000/results`
2. Enter admin credentials when prompted
3. View real-time analytics and records

### **Dashboard Features**

#### **Summary Cards**
- **PAUSE**: Count of customers who paused emails
- **BBAU**: Count of customers moved to international list
- **UNSUBSCRIBE**: Count of customers who unsubscribed

#### **CSV Downloads**
- Click **Download CSV** under any summary card
- Downloads filtered records for that action type
- Files named: `pause_records_2025-05-28.csv`

#### **Records Table**
- Shows all customer actions with timestamps
- Sorted by date (newest first)
- Displays: Date, Email, Action
- All times in Sydney Australia timezone

#### **Clear Records (Hidden Feature)**
1. Click on "Email Processing Results" title
2. Clear button appears
3. Confirms before deleting all records
4. Page refreshes to show empty state

---

## 🌐 Customer Interface

### **URL Format**
```
http://your-domain.com/?email=CUSTOMER_EMAIL
```

### **Customer Actions**
1. **Pause Sale Emails**: Temporarily skip current sale emails
2. **I'm Outside North America**: Move to international email list
3. **Unsubscribe Forever**: Remove from all email communications

### **Integration with Customer.io Emails**
```html
<!-- Add to your email templates -->
<a href="https://your-app.com/?email={{ customer.email }}">
  Manage Email Preferences
</a>
```

---

## 🗄️ Database & Data Management

### **SQLite Database**
- **File**: `email_processing.db`
- **Table**: `email_processing_records`
- **Columns**: id, timestamp, email, action

### **Data Retention**
- Records stored indefinitely unless manually cleared
- Admin can clear all records via dashboard
- Automatic logging of all customer actions

### **Backup Recommendations**
```bash
# Backup database
cp email_processing.db email_processing_backup_$(date +%Y%m%d).db

# Restore database
cp email_processing_backup_20250528.db email_processing.db
```

---

## 🚀 Deployment

### **Production Environment Variables**
```bash
CUSTOMERIO_SITE_ID=your_production_site_id
CUSTOMERIO_API_KEY=your_production_api_key
ADMIN_USERNAME=your_admin_email
ADMIN_PASSWORD=your_secure_password
PORT=8080
```

### **Fly.io Deployment (Recommended)**
```bash
# Quick deploy
./deploy.sh

# Manual setup
flyctl auth login
flyctl apps create your-app-name
flyctl secrets set CUSTOMERIO_SITE_ID=xxx CUSTOMERIO_API_KEY=xxx
flyctl deploy
```

### **Docker Deployment**
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/views ./views
EXPOSE 3000
CMD ["./main"]
```

---

## 🔒 Security Features

- **HTTP Basic Authentication**: Protects admin dashboard
- **Environment-based Credentials**: No hardcoded passwords
- **Input Validation**: Sanitizes customer email inputs
- **HTTPS Ready**: Secure API communications
- **Confirmation Dialogs**: Prevents accidental data deletion

---

## 📝 API Endpoints

### **Public Endpoints**
- `GET /` - Customer email preference interface
- `GET /ping` - Health check endpoint

### **Protected Endpoints** (Require Authentication)
- `GET /results` - Admin dashboard
- `GET /results/csv/:action` - Download CSV for specific action
- `POST /results/clear` - Clear all database records

---

## 🐛 Troubleshooting

### **Common Issues**

#### **Application Won't Start**
```bash
# Check environment variables
cat .env

# Verify Go installation
go version

# Check port availability
lsof -i :3000
```

#### **Authentication Issues**
- Verify `ADMIN_USERNAME` and `ADMIN_PASSWORD` in `.env`
- Clear browser cache/cookies
- Try incognito/private browsing mode

#### **Customer.io Integration Issues**
- Verify `CUSTOMERIO_SITE_ID` and `CUSTOMERIO_API_KEY`
- Check Customer.io dashboard for API key permissions
- Monitor application logs: `tail -f app.log`

#### **Database Issues**
```bash
# Check database file exists
ls -la email_processing.db

# Check database permissions
chmod 644 email_processing.db
```

---

## 📊 Monitoring & Logs

### **Application Logs**
- **File**: `app.log` (development) or stdout (production)
- **Format**: Timestamp + Source + Message
- **Levels**: INFO, ERROR, CRITICAL

### **Log Monitoring**
```bash
# Watch logs in real-time
tail -f app.log

# Search for errors
grep ERROR app.log

# Check recent activity
tail -100 app.log
```

---

## 🔄 Customer.io Workflow Integration

### **Segment Configuration**
Create segments in Customer.io based on attributes:

```
Active Email Recipients = 
  Related to BBUS entity 
  AND unsubscribed ≠ true 
  AND paused ≠ true
```

### **Campaign Exit Conditions**
Customers automatically exit campaigns when:
- `paused` attribute is set to `true`
- `unsubscribed` attribute is set to `true`
- Relationship to BBUS entity is removed

### **Testing Workflow**
1. Send test email with preference link
2. Test each action (pause, international, unsubscribe)
3. Verify attributes in Customer.io dashboard
4. Confirm campaign exit behavior

---

## 📞 Support & Maintenance

### **Regular Maintenance**
- Monitor disk space for database growth
- Review logs for errors or unusual activity
- Update admin credentials periodically
- Backup database regularly

### **Performance Monitoring**
- Monitor response times via `/ping` endpoint
- Check database query performance
- Monitor memory usage during high traffic

### **Getting Help**
- Check application logs first: `tail -f app.log`
- Verify environment configuration
- Test with curl commands
- Review Customer.io dashboard for API issues

---

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

**Note**: This application handles customer email preferences and must comply with email marketing regulations (CAN-SPAM, GDPR, etc.) in your jurisdiction. Always provide clear unsubscribe mechanisms and honor customer preferences promptly.