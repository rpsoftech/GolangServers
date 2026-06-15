CREATE TABLE
    `ItemGroup` (
        `itemGroupId` INT NOT NULL,
        `itemGroup` VARCHAR(45) NOT NULL,
        `itemPrintName` VARCHAR(45) NULL,
        `itemGroupUnitId` INT NOT NULL,
        PRIMARY KEY (`itemGroupId`)
    );

CREATE TABLE
    `ItemUnit` (
        `itemUnitId` INT NOT NULL,
        `itemUnit` VARCHAR(45) NOT NULL,
        `itemDecimal` INT (1) NULL DEFAULT 0,
        PRIMARY KEY (`itemUnitId`),
        UNIQUE INDEX `itemUnit_UNIQUE` (`itemUnit` ASC) VISIBLE
    );

ALTER TABLE `ItemGroup` ADD INDEX `IgroupUnitToUnitID_idx` (`itemGroupUnitId` ASC) VISIBLE;

ALTER TABLE `ItemGroup` ADD CONSTRAINT `IgroupUnitToUnitID` FOREIGN KEY (`itemGroupUnitId`) REFERENCES `ItemUnit` (`itemUnitId`) ON DELETE RESTRICT ON UPDATE CASCADE;

CREATE TABLE
    `ItemMaster` (
        `itemId` INT NOT NULL,
        `itemName` VARCHAR(45) NULL,
        `iGroupId` INT (3) NULL,
        `itemPrintName` VARCHAR(45) NULL,
        `iUnitId` INT (3) NULL,
        `itemTagPrefix` VARCHAR(5) NULL,
        PRIMARY KEY (`itemId`),
        UNIQUE INDEX `itemName_UNIQUE` (`itemName` ASC) VISIBLE,
        INDEX `ItemMast1_idx` (`iGroupId` ASC) VISIBLE,
        INDEX `ItemMast2_idx` (`iUnitId` ASC) VISIBLE,
        CONSTRAINT `ItemMast1` FOREIGN KEY (`iGroupId`) REFERENCES `ItemGroup` (`itemGroupId`) ON DELETE RESTRICT ON UPDATE CASCADE,
        CONSTRAINT `ItemMast2` FOREIGN KEY (`iUnitId`) REFERENCES `ItemUnit` (`itemUnitId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );

CREATE TABLE
    `Stamp` (
        `stampId` INT NOT NULL,
        `STAMP` VARCHAR(45) NULL,
        `tunch` DECIMAL(8, 2) NULL,
        `stockTunch` DECIMAL(8, 2) NULL,
        `sellTunch` DECIMAL(8, 2) NULL DEFAULT 0,
        PRIMARY KEY (`stampId`)
    );

CREATE TABLE
    `rpg_jwelly_test`.`ItemsTag` (
        `itemTagId` INT NOT NULL,
        `itemTag` VARCHAR(45) NOT NULL,
        `itemVTagId` INT NULL,
        `tItemId` INT NULL,
        `tagCreatedDate` DATE NULL,
        PRIMARY KEY (`itemTagId`),
        UNIQUE INDEX `itemTag_UNIQUE` (`itemTag` ASC) VISIBLE,
        UNIQUE INDEX `combo_unique` (`itemVTagId` ASC, `tItemId` ASC) VISIBLE,
        INDEX `itemtag_item_idx` (`tItemId` ASC) VISIBLE,
        CONSTRAINT `itemtag_item` FOREIGN KEY (`tItemId`) REFERENCES `rpg_jwelly_test`.`ItemMaster` (`itemId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );

CREATE TABLE
    `rpg_jwelly_test`.`ItemTagVariation` (
        `tagVariationId` INT NOT NULL AUTO_INCREMENT,
        `vTagId` INT NULL,
        `vStampId` INT NULL,
        `vGrossWt` DECIMAL(10, 4) NULL,
        `vLessWeight` DECIMAL(10, 4) NULL,
        `vNetWt` DECIMAL(10, 4) NULL,
        `vStatus` BIT (1) NULL,
        `vTunch` DECIMAL(6, 2) NULL,
        `vWstg` DECIMAL(6, 2) NULL,
        `vSellTunch` DECIMAL(6, 2) NULL,
        `vSellWstg` DECIMAL(6, 2) NULL,
        `vKarigarCode` VARCHAR(40) NULL,
        PRIMARY KEY (`tagVariationId`),
        INDEX `TagVariation_1_idx` (`vTagId` ASC) INVISIBLE,
        INDEX `TagVariation_2_idx` (`vStampId` ASC) VISIBLE,
        UNIQUE INDEX `TagVariation_3` (`vTagId` ASC, `vStampId` ASC) INVISIBLE,
        CONSTRAINT `TagVariation_1` FOREIGN KEY (`vTagId`) REFERENCES `rpg_jwelly_test`.`ItemsTag` (`itemTagId`) ON DELETE RESTRICT ON UPDATE CASCADE,
        CONSTRAINT `TagVariation_2` FOREIGN KEY (`vStampId`) REFERENCES `rpg_jwelly_test`.`Stamp` (`stampId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );