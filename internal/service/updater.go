package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"volcengine-updater/internal/config"
	"volcengine-updater/internal/models"

	"github.com/volcengine/volcengine-go-sdk/service/vpc"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// CheckAndUpdate is the main entry point for the scheduled task
func CheckAndUpdate() {
	settings := config.GetSettings()
	if settings.AccessKey == "" || settings.SecretKey == "" || settings.SecurityGroupID == "" {
		config.Log("WARNING", "Task skipped: Incomplete configuration (AK/SK/SG_ID missing)")
		return
	}

	config.Log("INFO", "Starting IP check...")

	currentIP := getCurrentIP(settings.IPServices)
	if currentIP == "" {
		config.Log("ERROR", "Failed to get current public IP, skipping check")
		return
	}

	updateSecurityGroup(settings, currentIP)
}

func getCurrentIP(servicesStr string) string {
	services := strings.Split(servicesStr, "\n")
	client := &http.Client{Timeout: 5 * time.Second}

	for _, url := range services {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		resp, err := client.Get(url)
		if err != nil {
			config.Log("WARNING", fmt.Sprintf("Failed to get IP from %s: %v", url, err))
			continue
		}
		

		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			ip := strings.TrimSpace(string(body))
			if ip != "" {
				config.Log("INFO", fmt.Sprintf("Current Public IP: %s (Source: %s)", ip, url))
				return ip
			}
		} else {
			resp.Body.Close()
		}
	}
	return ""
}

func updateSecurityGroup(settings *models.Settings, currentIP string) {
	conf := volcengine.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(settings.AccessKey, settings.SecretKey, "")).
		WithRegion(settings.Region)

	sess, err := session.NewSession(conf)
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("Failed to create session: %v", err))
		return
	}

	vpcClient := vpc.New(sess)

	// Get current rules
	input := &vpc.DescribeSecurityGroupAttributesInput{
		SecurityGroupId: volcengine.String(settings.SecurityGroupID),
	}

	output, err := vpcClient.DescribeSecurityGroupAttributes(input)
	if err != nil {
		config.Log("ERROR", fmt.Sprintf("Failed to describe security group: %v", err))
		return
	}

	var existingRule *vpc.PermissionForDescribeSecurityGroupAttributesOutput

	// Find existing SSH rule
	for _, perm := range output.Permissions {
		if volcengine.StringValue(perm.Direction) == "ingress" &&
			(strings.EqualFold(volcengine.StringValue(perm.Protocol), "tcp") || strings.EqualFold(volcengine.StringValue(perm.Protocol), "all")) &&
			int(volcengine.Int64Value(perm.PortStart)) <= settings.SSHPort &&
			int(volcengine.Int64Value(perm.PortEnd)) >= settings.SSHPort {
			existingRule = perm
			break
		}
	}

	if existingRule != nil {
		currentCidr := volcengine.StringValue(existingRule.CidrIp)
		existingIP := strings.Split(currentCidr, "/")[0]
		
		config.Log("INFO", fmt.Sprintf("Found existing SSH rule IP: %s", existingIP))

		if existingIP == currentIP {
			config.Log("INFO", "✓ IP address unchanged, no update needed")
			return
		}

		// Revoke old rule
		config.Log("INFO", fmt.Sprintf("Revoking old SSH rule: %s", currentCidr))
		_, err := vpcClient.RevokeSecurityGroupIngress(&vpc.RevokeSecurityGroupIngressInput{
			SecurityGroupId: volcengine.String(settings.SecurityGroupID),
			Protocol:        existingRule.Protocol,
			PortStart:       volcengine.Int64(int64(settings.SSHPort)),
			PortEnd:         volcengine.Int64(int64(settings.SSHPort)),
			CidrIp:          existingRule.CidrIp,
			Policy:          existingRule.Policy,
		})
		if err != nil {
			config.Log("WARNING", fmt.Sprintf("Failed to revoke old rule (continuing): %v", err))
		}
	} else {
		config.Log("INFO", "No existing SSH rule found, adding new one")
	}

	// Authorize new rule
	newCidr := fmt.Sprintf("%s/32", currentIP)
	config.Log("INFO", fmt.Sprintf("Adding new SSH rule: %s", newCidr))
	
	_, err = vpcClient.AuthorizeSecurityGroupIngress(&vpc.AuthorizeSecurityGroupIngressInput{
		SecurityGroupId: volcengine.String(settings.SecurityGroupID),
		Protocol:        volcengine.String("TCP"),
		PortStart:       volcengine.Int64(int64(settings.SSHPort)),
		PortEnd:         volcengine.Int64(int64(settings.SSHPort)),
		CidrIp:          volcengine.String(newCidr),
		Policy:          volcengine.String("accept"),
		Priority:        volcengine.Int64(1),
		Description:     volcengine.String("SSH access - Auto updated by Go script"),
	})

	if err != nil {
		config.Log("ERROR", fmt.Sprintf("Failed to authorize new rule: %v", err))
	} else {
		config.Log("INFO", fmt.Sprintf("✓ Security group updated: Allow %s access to port %d", newCidr, settings.SSHPort))
	}
}
