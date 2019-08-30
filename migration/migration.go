package migration

import (
	"strings"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/models"
	"github.com/jinzhu/gorm"
	gormigrate "gopkg.in/gormigrate.v1"
)

// Migration contains all of the incremental migrations that the database
// requires to keep its schema and models up to date with current GoBroker
// source code.
func Migration(db *gorm.DB) *gormigrate.Gormigrate {
	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// initial migration
		{
			ID: "201804241345",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&models.Account{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Owner{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.OwnerDetails{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Position{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.ALEStatus{}).Error; err != nil {
					return err
				}
				for _, topic := range apex.ALETopics {
					if err := tx.Create(
						&models.ALEStatus{
							Topic:     string(topic),
							Watermark: uint64(0),
						}).Error; err != nil {
						return err
					}
				}
				if err := tx.AutoMigrate(&models.Transfer{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.ACHRelationship{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Investigation{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Affiliate{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Administrator{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.DayPLSnapshot{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.DocumentRequest{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Snap{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Asset{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Fundamental{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.AccessKey{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Order{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Execution{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.TrustedContact{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.TradeFailure{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.BatchError{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.BatchMetric{}).Error; err != nil {
					return err
				}
				return tx.AutoMigrate(&models.HermesFailure{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "201804301548",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE owners ADD COLUMN email_confirmed_at TIMESTAMP WITH TIME ZONE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Owner{}).DropColumn("email_confirmed_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201804301621",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Asset{}).DropColumn("deleted_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE assets ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201805012308",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts RENAME COLUMN amount_tradable TO cash").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Exec("ALTER TABLE accounts RENAME COLUMN amount_withdrawable TO cash_withdrawable").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts RENAME COLUMN cash TO amount_tradable").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Exec("ALTER TABLE accounts RENAME COLUMN cash_withdrawable TO amount_withdrawable").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201805021032",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE transfers ADD COLUMN batch_processed_at date").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Transfer{}).DropColumn("batch_processed_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201805041152",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE positions RENAME COLUMN shares TO qty").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE positions RENAME COLUMN qty TO shares").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201805070916",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.MarginCall{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("margin_calls").Error
			},
		},
		{
			ID: "201805070918",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("DROP INDEX uix_orders_client_order_id").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return tx.AutoMigrate(&models.Order{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("DROP INDEX idx_client_order_id_account").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201805071326",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts ADD COLUMN protect_pattern_day_trader BOOLEAN NOT NULL DEFAULT TRUE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Model(&models.Account{}).DropColumn("protect_pattern_day_trader").Error
			},
		},
		{
			ID: "201805081102",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.OwnerDetails{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Model(&models.OwnerDetails{}).DropColumn("visa_expiration_date").Error
			},
		},
		{
			ID: "201805081438",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts ADD COLUMN marked_pattern_day_trader_at DATE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Model(&models.Account{}).DropColumn("marked_pattern_day_trader_at").Error
			},
		},
		{
			ID: "201805151251",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE executions ALTER COLUMN broker_exec_id SET NOT NULL").Error; err != nil {
					return err
				}
				if err := tx.Exec("ALTER TABLE executions ALTER COLUMN transaction_time SET NOT NULL").Error; err != nil {
					return err
				}
				if err := tx.Exec("CREATE UNIQUE INDEX idx_execution_broker_exec_id_transaction_time ON executions (broker_exec_id, transaction_time)").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE executions ALTER COLUMN broker_exec_id SET NULL").Error; err != nil {
					return err
				}
				if err := tx.Exec("ALTER TABLE executions ALTER COLUMN transaction_time SET NULL").Error; err != nil {
					return err
				}
				if err := tx.Exec("DROP INDEX idx_execution_broker_exec_id_transaction_time").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201805161602",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE positions ADD COLUMN marked_for_split_at DATE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Model(&models.Position{}).DropColumn("marked_for_split_at").Error
			},
		},
		{
			ID: "201805181322",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE positions ALTER COLUMN entry_price SET NOT NULL").Error; err != nil {
					return err
				}
				if err := tx.Exec("ALTER TABLE positions ALTER COLUMN entry_timestamp SET NOT NULL").Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE positions ALTER COLUMN entry_price SET NULL").Error; err != nil {
					return err
				}
				if err := tx.Exec("ALTER TABLE positions ALTER COLUMN entry_timestamp SET NULL").Error; err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "201805171741",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.Cash{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("cashes").Error
			},
		},
		{
			ID: "201805291414",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE owners ADD COLUMN reset_token TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Owner{}).DropColumn("reset_token").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201806121108",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec("UPDATE orders SET status = 'canceled' WHERE status = 'cancelled'").Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Exec("UPDATE orders SET status = 'cancelled' WHERE status = 'canceled'").Error
			},
		},
		{
			ID: "201806121408",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.EmailVerificationCode{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("email_verification_codes").Error
			},
		},
		{
			ID: "201806151146",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE transfers ADD COLUMN balance_validated boolean").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Transfer{}).DropColumn("balance_validated").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201806141549",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.ACHRelationship{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE ach_relationships DROP COLUMN mask").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE ach_relationships DROP COLUMN nickname").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
		},
		{
			ID: "20180618741",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&models.Order{}).Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&models.Execution{}).Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE orders DROP COLUMN fee").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE executions DROP COLUMN fee_sec, DROP COLUMN fee_misc, DROP COLUMN fee1, DROP COLUMN fee2, DROP COLUMN fee3, DROP COLUMN fee4, DROP COLUMN fee5").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201806201104",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&models.Dividend{}).Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("dividends").Error
			},
		},
		{
			ID: "201807021439",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE orders ALTER COLUMN client_order_id TYPE varchar(50)").Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE orders ALTER COLUMN client_order_id TYPE uuid").Error; err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "201807021625",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts ALTER COLUMN currency SET DEFAULT 'USD'").Error; err != nil {
					return err
				}
				if err := tx.Exec("UPDATE accounts SET currency = 'USD' WHERE currency IS NULL").Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts ALTER COLUMN currency DROP DEFAULT").Error; err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "201807271309",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE orders ADD COLUMN is_correction BOOLEAN NOT NULL DEFAULT 'f'").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Order{}).DropColumn("is_correction").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201807291327",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE orders ALTER COLUMN order_capacity SET DEFAULT 'agency'").Error; err != nil {
					return err
				}
				return tx.Exec("UPDATE orders SET order_capacity = 'agency' where order_capacity = 'A'").Error
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE orders ALTER COLUMN order_capacity SET DEFAULT 'A'").Error; err != nil {
					return err
				}
				return tx.Exec("UPDATE orders SET order_capacity = 'A' where order_capacity = 'agency'").Error
			},
		},
		{
			ID: "201807301405",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.CorporateAction{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("corporate_actions").Error
			},
		},
		{
			ID: "201808011343",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE transfers ADD COLUMN type TEXT NOT NULL DEFAULT 'ach'").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE transfers ALTER COLUMN relationship_id DROP NOT NULL").Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Owner{}).DropColumn("type").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Exec("ALTER TABLE transfers ALTER COLUMN relationship_id SET NOT NULL").Error; err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "201808011501",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE assets ADD COLUMN shortable boolean NOT NULL DEFAULT 'f'").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Asset{}).DropColumn("shortable").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil

			},
		},
		{
			ID: "201808011603",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.PaperAccount{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("paper_accounts").Error
			},
		},
		{
			ID: "201808031043",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN employer_address TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN function TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN nasdaq_agreement_signed_at TIMESTAMP WITH TIME ZONE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN nyse_agreement_signed_at TIMESTAMP WITH TIME ZONE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("employer_address").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("function").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("nasdaq_agreement_signed_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("nyse_agreement_signed_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}

				return nil
			},
		},
		{
			ID: "201808221233",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.AdminNote{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("admin_notes").Error
			},
		},
		{
			ID: "201808241709",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.AdminEmail{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("admin_emails").Error
			},
		},
		{
			ID: "201808301610",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE admin_notes ADD COLUMN admin_id UUID").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE admin_emails ADD COLUMN admin_id UUID").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.AdminNote{}).DropColumn("admin_id").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.AdminEmail{}).DropColumn("admin_id").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201809051307",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN replaced_at TIMESTAMP WITH TIME ZONE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN replaced_by INTEGER").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN replaces INTEGER").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN approved_by TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("DROP INDEX uix_owner_details_owner_id").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("replaced_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("replaced_by").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("replaces").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("approved_by").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Exec("CREATE UNIQUE INDEX uix_owner_details_owner_id ON owner_details (owner_id)").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201809121328",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN approved_by TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN approved_at TIMESTAMP WITH TIME ZONE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("approved_by").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("approved_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201809201225",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE owner_details ADD COLUMN assigned_admin_id UUID").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.OwnerDetails{}).DropColumn("assigned_admin_id").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201809211238",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Account{}).DropColumn("email_confirmed_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.Account{}).DropColumn("hash_password").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.Account{}).DropColumn("salt").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.Account{}).DropColumn("reset_token").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts ADD COLUMN email_confirmed_at TIMESTAMP WITH TIME ZONE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE accounts ADD COLUMN hash_password BYTEA").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE accounts ADD COLUMN salt BYTEA").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE accounts ADD COLUMN reset_token TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201809211704",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts ADD COLUMN cognito_id UUID").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Account{}).DropColumn("cognito_id").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201809250907",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE ach_relationships ADD COLUMN hash_bank_info BYTEA").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE ach_relationships ADD COLUMN expires_at TIMESTAMP WITH TIME ZONE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE transfers ADD COLUMN expires_at TIMESTAMP WITH TIME ZONE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.ACHRelationship{}).DropColumn("hash_bank_info").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.ACHRelationship{}).DropColumn("expires_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.Transfer{}).DropColumn("expires_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201809260848",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE ach_relationships ADD COLUMN apex_id TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.ACHRelationship{}).DropColumn("apex_id").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201809270910",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE affiliates ADD COLUMN type TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Affiliate{}).DropColumn("type").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201810221707",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE assets ADD COLUMN cusip_old TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE assets ADD COLUMN symbol_old TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Asset{}).DropColumn("cusip_old").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.Asset{}).DropColumn("symbol_old").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201810311538",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE ach_relationships ADD COLUMN failed_attempts INT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE ach_relationships ADD COLUMN reason TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.ACHRelationship{}).DropColumn("failed_attempts").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.ACHRelationship{}).DropColumn("reason").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201810311659",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE transfers ADD COLUMN reason TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE transfers ADD COLUMN reason_code TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Transfer{}).DropColumn("reason").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.Transfer{}).DropColumn("reason_code").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201811081532",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE affiliates ADD COLUMN company_symbol TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Affiliate{}).DropColumn("company_symbol").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201811091451",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE ach_relationships ADD COLUMN micro_deposit_id TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE ach_relationships ADD COLUMN micro_deposit_status TEXT").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.ACHRelationship{}).DropColumn("micro_deposit_id").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.ACHRelationship{}).DropColumn("micro_deposit_status").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201812101327",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts ADD COLUMN marked_risky_transfers_at DATE").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Exec("ALTER TABLE accounts RENAME COLUMN transfers_blocked TO risky_transfers").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Account{}).DropColumn("marked_risky_transfers_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Exec("ALTER TABLE accounts RENAME COLUMN risky_transfers TO transfers_blocked").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201812220058",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE accounts ADD COLUMN trade_suspended_by_user bool NOT NULL DEFAULT false").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Account{}).DropColumn("trade_suspended_by_user").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
		{
			ID: "201901011713",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Order{}).AddIndex("idx_account", "account").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Model(&models.Order{}).AddIndex("idx_submitted_at", "submitted_at").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Model(&models.Order{}).AddIndex("idx_order_status", "status").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				if err := tx.Model(&models.Position{}).AddIndex("idx_position_status", "status").Error; err != nil && !strings.Contains(err.Error(), "already exists") {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Model(&models.Order{}).RemoveIndex("idx_account").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.Order{}).RemoveIndex("idx_submitted_at").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.Order{}).RemoveIndex("idx_order_status").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				if err := tx.Model(&models.Position{}).RemoveIndex("idx_position_status").Error; err != nil && !strings.Contains(err.Error(), "does not exist") {
					return err
				}
				return nil
			},
		},
	})
}
