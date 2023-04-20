package runner

import "time"

type BackupDefinitionBuilder struct {
	backupDefinition BackupDefinition
}

func (b BackupDefinitionBuilder) Operate(url string) BackupDefinitionBuilder {
	b.backupDefinition.operateURL = url
	return b
}

func (b BackupDefinitionBuilder) Optimize(url string) BackupDefinitionBuilder {
	b.backupDefinition.optimizeURL = url
	return b
}

func (b BackupDefinitionBuilder) Tasklist(url string) BackupDefinitionBuilder {
	b.backupDefinition.tasklistURL = url
	return b
}

func (b BackupDefinitionBuilder) Zeebe(url string) BackupDefinitionBuilder {
	b.backupDefinition.zeebeURL = url
	return b
}

func (b BackupDefinitionBuilder) ZeebeIndexPrefix(prefix string) BackupDefinitionBuilder {
	b.backupDefinition.zeebeIndexPrefix = prefix
	return b
}

func (b BackupDefinitionBuilder) Elastic(url, snapshotRepositoryName string) BackupDefinitionBuilder {
	b.backupDefinition.elasticURL = url
	b.backupDefinition.backupRepositoryName = snapshotRepositoryName
	return b
}

func (b BackupDefinitionBuilder) Build() BackupDefinition {
	b.backupDefinition.backupID = time.Now().Unix()
	if b.backupDefinition.zeebeIndexPrefix == "" {
		b.backupDefinition.zeebeIndexPrefix = "zeebe-record*"
	}
	return b.backupDefinition
}

func NewBackupDefinitionBuilder() BackupDefinitionBuilder {
	return BackupDefinitionBuilder{}
}
