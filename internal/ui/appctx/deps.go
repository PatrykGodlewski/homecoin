package appctx

import (
	authuc "github.com/godlew/homecoin/internal/usecase/auth"
	balanceuc "github.com/godlew/homecoin/internal/usecase/balance"
	budgetuc "github.com/godlew/homecoin/internal/usecase/budget"
	categoryuc "github.com/godlew/homecoin/internal/usecase/category"
	expenseuc "github.com/godlew/homecoin/internal/usecase/expense"
	householduc "github.com/godlew/homecoin/internal/usecase/household"
	piggybankuc "github.com/godlew/homecoin/internal/usecase/piggybank"
)

type Application struct {
	Register       *authuc.RegisterUseCase
	Login          *authuc.LoginUseCase
	Me             *authuc.MeUseCase
	CreateHH         *householduc.CreateUseCase
	JoinHH           *householduc.JoinUseCase
	GetHH            *householduc.GetUseCase
	GetMineHH        *householduc.GetMineUseCase
	AddExpense       *expenseuc.AddUseCase
	ListExpenses     *expenseuc.ListUseCase
	GetBalances      *balanceuc.GetUseCase
	SimplifyBal      *balanceuc.SimplifyUseCase
	UsageBudget      *budgetuc.UsageUseCase
	CreateBudget     *budgetuc.CreateUseCase
	ListCategories   *categoryuc.ListUseCase
	CreatePiggy      *piggybankuc.CreateUseCase
	Contribute       *piggybankuc.ContributeUseCase
	ListPiggy        *piggybankuc.ListUseCase
}

var App *Application
