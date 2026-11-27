package appenv

// UsesS3 reports whether uploads are stored in an S3-compatible bucket.
func (e *Env) UsesS3() bool {
	if e == nil {
		return false
	}

	return e.StorageBackend == StorageBackendS3
}

func (e *Env) missingS3Settings() []string {
	if !e.UsesS3() {
		return nil
	}

	var missing []string
	if e.S3Endpoint == "" {
		missing = append(missing, "S3_ENDPOINT")
	}
	if e.S3AccessKey == "" {
		missing = append(missing, "S3_ACCESS_KEY")
	}
	if e.S3SecretKey == "" {
		missing = append(missing, "S3_SECRET_KEY")
	}
	if e.S3Bucket == "" {
		missing = append(missing, "S3_BUCKET")
	}

	return missing
}
