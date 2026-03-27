# 🚀 GDriveDisk: Google Drive-Based RAM Disk  

GDriveDisk is a virtual ROM disk that mounts **Google Drive as a high-speed memory drive**, allowing temporary storage that syncs with Google Drive in real-time.  

## 📌 Features  
✅ **Mount Google Drive as a Virtual File System (FUSE)**  
✅ **Fast Read/Write Operations with Caching**  
✅ **Bloom Filters for Quick File Lookup**  
✅ **Redis Cache to Reduce API Calls**  
✅ **Adaptive Prefetching Algorithm**  

---

## 🛠️ Tech Stack  
- **GoLang** (Concurrency & Performance)  
- **Google Drive API** (Cloud File Storage)  
- **FUSE (Filesystem in Userspace)** (Virtual Drive)  
- **Redis** (Fast Caching)  
- **Bloom Filters** (Quick Lookup)  

----

## 📌 Installation & Setup  

### 1️⃣ **Install Dependencies**  
```bash
sudo apt install golang
go version
go get -u bazil.org/fuse
go get -u golang.org/x/oauth2
go get -u google.golang.org/api/drive/v3
go get -u github.com/go-redis/redis/v8
go get -u github.com/bits-and-blooms/bloom/v3
```

### 2️⃣ **Setup Google Drive API**  
1. Go to **Google Cloud Console**  
2. Enable **Google Drive API**  
3. Create **OAuth 2.0 credentials**  
4. Download `credentials.json` and place it in the project root  

---

## 🚀 Run the Project  

### **Authenticate with Google Drive**
```bash
go run auth.go
```

### **Mount Google Drive as Virtual RAM Disk**
```bash
go run main.go
```
> ✅ Drive will be mounted at `/mnt/gdrive`  

---

## 🗂️ Usage  

### **Read File from Google Drive**  
```go
content, err := readFile("file_id", driveService)
if err != nil {
    log.Fatal(err)
}
fmt.Println("File Content:", string(content))
```

### **Write File to Google Drive**  
```go
err := writeFile("test.txt", []byte("Hello from RAM Disk!"), driveService)
if err != nil {
    log.Fatal(err)
}
fmt.Println("File Uploaded Successfully!")
```

### **Cache Files in Redis**
```go
cacheFile("test.txt", []byte("Cached Content"))
data, err := getCachedFile("test.txt")
fmt.Println("Cached Data:", string(data))
```

---

## 📌 Optimizations  

### 🔹 **Bloom Filters for Fast Lookups**  
```go
addToBloomFilter("test.txt")
if isFileInCache("test.txt") {
    fmt.Println("File exists in cache!")
}
```

### 🔹 **Redis Caching**  
- Frequently accessed files are stored in **Redis**  
- Reduces **API calls** and speeds up access  

### 🔹 **Adaptive Prefetching Algorithm**  
- Uses access patterns to **predict next files**  
- Loads them into **memory for faster access**  

---

## 🤝 Contributing  
Want to improve GDriveDisk? Feel free to submit **PRs & Issues**!  

---

## 📜 License  
This project is licensed under **MIT License**.  

---

## ⭐ Support  
If you like this project, **give it a star ⭐** and contribute!  
